//! Main application loop

#![allow(dead_code)]

use std::collections::HashMap;
use std::hash::DefaultHasher;
use std::hash::Hash;
use std::hash::Hasher;
use std::sync::Arc;

use datafusion::arrow::datatypes::{Field, Schema};
use datafusion::arrow::record_batch::RecordBatch;
use datafusion::prelude::SessionContext;
use tokio::sync::mpsc;
use tracing::{error, info, instrument, warn};

use crate::chan;
use crate::core;

/// Compares two Arrow Schemas to identify structural changes.
///
/// This function performs a field-by-field comparison to detect new columns,
/// removed columns, and data type or nullability changes.
///
/// ### Logic Flow:
/// 1. Maps both schemas into `HashMap`s for O(1) lookups by field name.
/// 2. Iterates through the `new` schema to identify **Created** or **Updated** fields.
/// 3. Iterates through the `old` schema to identify **Deleted** fields.
///
/// ### Complexity:
/// **O(N + M)**, where N and M are the number of fields in the old and new schemas.
///
/// ### Parameters:
/// * `old`: The current schema of the target table (the "Baseline").
/// * `new`: The schema inferred from the incoming batch (the "Desired State").
#[instrument]
pub fn compare_schemas(old: &Schema, new: &Schema) -> core::SchemaDiff {
    let old_fields: HashMap<String, &Field> = old
        .fields()
        .iter()
        .map(|field| (field.name().clone(), field.as_ref()))
        .collect();

    let new_fields: HashMap<String, &Field> = new
        .fields()
        .iter()
        .map(|field| (field.name().clone(), field.as_ref()))
        .collect();

    let mut created: Vec<Field> = Vec::new();
    let mut updated: Vec<(Field, Field)> = Vec::new();
    for (name, new_field) in &new_fields {
        match old_fields.get(name) {
            Some(old_field) => {
                if !field_props_are_equal(old_field, new_field) {
                    updated.push(((**old_field).clone(), (**new_field).clone()));
                }
            }
            None => created.push((**new_field).clone()),
        }
    }

    let mut deleted: Vec<Field> = Vec::new();
    for (name, old_field) in &old_fields {
        if !new_fields.contains_key(name) {
            deleted.push((**old_field).clone());
        }
    }

    core::SchemaDiff {
        created,
        updated,
        deleted,
    }
}

#[inline]
fn field_props_are_equal(l: &Field, r: &Field) -> bool {
    l.is_nullable() == r.is_nullable() && l.data_type() == r.data_type()
}

#[instrument]
fn get_schema_hash(schema: &Schema) -> u64 {
    let mut hasher = DefaultHasher::new();
    for field in schema.fields() {
        field.hash(&mut hasher);
    }
    hasher.finish()
}

#[instrument(skip(ctx, database, metadata, records), fields(table = %metadata.table, partition = %metadata.partition, position = %metadata.position))]
async fn write_batch(
    ctx: &SessionContext,
    worker: usize,
    database: &dyn core::Database,
    metadata: &core::SourceMetadata,
    records: RecordBatch,
) -> Result<(), core::WriterError> {
    let size = records.num_rows();
    info!("processing records batch");

    let df = match ctx.read_batch(records) {
        Ok(df) => df,
        Err(err) => {
            return Err(core::WriterError::DataFusion(err));
        }
    };

    // TODO: apply SQL transformation.

    let writer = database.get_writer(&metadata).await?;
    let old_schema = writer.schema().await?;
    let old_schema_hash = get_schema_hash(&old_schema);
    let new_schema = df.schema().as_arrow();
    let new_scheme_hash = get_schema_hash(new_schema);

    if old_schema_hash != new_scheme_hash {
        info!("detected schema missmatch");
        let diff = compare_schemas(new_schema.as_ref(), old_schema.as_ref());

        info!("altering schema");
        writer.alter(&diff).await?;
    }

    // TODO: Register new schema version in the data catalog. check if present via new_schema_hash

    writer.upsert(&df).await?;

    info!(size = size, "batch has been commited");
    Ok(())
}

pub async fn run(
    reader: &dyn core::Reader,
    database: &Arc<dyn core::Database>,
    dlq_writer: &dyn core::DlqWriter,
) -> Result<(), core::AppError> {
    info!("starting main loop");

    let (tx_dlq, mut rx_dlq) = mpsc::channel::<core::DlqMessage>(1000);
    let (tx_ack, mut rx_ack) = mpsc::channel::<core::SourceMetadata>(1000);
    let (tx_rec, rx_rec) = chan::make_channels::<core::SourceBatch>(1000, 8);

    for (i, rx) in rx_rec.into_iter().enumerate() {
        let tx_dlq = tx_dlq.clone();
        let tx_ack = tx_ack.clone();
        let database = database.clone();
        let ctx = SessionContext::new();

        tokio::spawn(async move {
            info!(worker = i, "worker started");

            let mut rx_local = rx;

            while let Some(batch) = rx_local.recv().await {
                let db = database.as_ref();
                if let Err(err) = write_batch(&ctx, i, db, &batch.metadata, batch.records).await {
                    error!(
                        worker = i,
                        error = %err,
                        table = batch.metadata.table,
                        partition = %batch.metadata.partition,
                        offset = %batch.metadata.position,
                        "batch processing failed",
                    );

                    match err {
                        core::WriterError::Data(e, records) => {
                            let message = core::DlqMessage {
                                metadata: batch.metadata.clone(),
                                payload: core::DlqPayload::Batch(records),
                                error_message: e.to_string(),
                            };

                            if let Err(_) = tx_dlq.send(message).await {
                                warn!(number = i, "dql channel closed, terminating worker");
                            }
                        }
                        _ => {
                            warn!(worker = i, "terminating worker");
                            break;
                        }
                    }
                }

                if let Err(_) = tx_ack.send(batch.metadata).await {
                    warn!(worker = i, "ack channel closed, terminating worker");
                    break;
                }
            }

            info!(worker = i, "worker terminated");
        });
    }

    loop {
        let dlq = chan::try_recv_batch(&mut rx_dlq, 1000);
        match dlq {
            Ok(dlq) => {
                if dlq.len() > 100 {
                    error!("too many dead letters");
                    // TODO: terminate main loop
                }
                // increment statistics
                dlq_writer.write(dlq).await?;
                println!("received dlq");
            }
            Err(mpsc::error::TryRecvError::Disconnected) => {
                warn!("dql channel disconnected, terminating main loop");
                break;
            }
            Err(mpsc::error::TryRecvError::Empty) => {}
        }

        let ack = chan::try_recv_batch(&mut rx_ack, 1000);
        match ack {
            Ok(ack) => {
                reader.commit(&ack).await?;
                // increment statistics
                println!("received ack");
            }
            Err(mpsc::error::TryRecvError::Disconnected) => {
                warn!("ack channel disconnected, terminating main loop");
                break;
            }
            Err(mpsc::error::TryRecvError::Empty) => {}
        }

        let result = reader.fetch().await;
        match result {
            Ok(fetched) => {
                if let Err(_) = chan::send_all(&tx_dlq, fetched.failure).await {
                    warn!("dlq channel closed, terminating main loop");
                    break;
                }

                if let Some(batch) = fetched.success {
                    info!(
                        table = batch.metadata.table,
                        partition = %batch.metadata.partition,
                        position = %batch.metadata.position,
                        size = batch.records.num_rows(),
                        "fetched records batch",
                    );

                    let i = batch.metadata.hash() % (tx_rec.len() as u64);
                    if let Err(_) = tx_rec[i as usize].send(batch).await {
                        warn!("record channel closed, terminating main loop");
                        break;
                    }
                }
            }
            Err(err) => {
                // TODO: Add proper error handling.
                error!(error = %err, "reader error");
            }
        }
    }

    info!("main loop terminated");
    Ok(())
}
