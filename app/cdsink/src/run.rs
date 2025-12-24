//! Main application loop

#![allow(dead_code)]

use std::collections::HashMap;

use datafusion::arrow::datatypes::{Field, Schema};

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
                if !are_equal(old_field, new_field) {
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
fn are_equal(l: &Field, r: &Field) -> bool {
    l.is_nullable() == r.is_nullable() && l.data_type() == r.data_type()
}

// pub async fn run(
//     reader: &dyn Reader,
//     tables: &[&dyn Table],
//     dlq_writer: &dyn DQLWriter,
// ) -> Result<(), AppError> {
//     info!("starting main loop");

//     let (mut tx_dlq, mut rx_dlq) = mpsc::channel::<DeadLetter>(100);
//     let (mut tx_ack, mut rx_ack) = mpsc::channel::<SourceMetadata>(100);

//     // spawn writers loops via tokio
//     // writer loop
//     // get next records batch
//     // convert to datafusion dataframe
//     // get table

//     loop {
//         let dlq = try_recv_batch(&mut rx_dlq, 100).await;
//         match dlq {
//             Ok(dlq) => {
//                 // increment statistics
//                 dlq_writer.write(dlq).await?;
//                 println!("received dlq");
//             }
//             Err(mpsc::error::TryRecvError::Disconnected) => {
//                 debug!("dql channel disconnected");
//                 break;
//             }
//             Err(mpsc::error::TryRecvError::Empty) => {}
//         }

//         // TODO: If to many dead letters then terminate

//         let ack = try_recv_batch(&mut rx_ack, 100).await;
//         match ack {
//             Ok(ack) => {
//                 reader.commit(&ack).await?;
//                 // increment statistics
//                 println!("received ack");
//             }
//             Err(mpsc::error::TryRecvError::Disconnected) => {
//                 debug!("ack channel disconnected");
//                 break;
//             }
//             Err(mpsc::error::TryRecvError::Empty) => {}
//         }

//         let result = reader.fetch().await;
//         match result {
//             Ok(Some(_)) => {
//                 println!("received records batch");
//             }
//             Ok(None) => {
//                 tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
//             }
//             Err(err) => {
//                 // TODO: Add proper error handling.
//                 error!(error = %err, "reader error");
//             }
//         }
//     }

//     info!("exiting main loop");
//     Ok(())
// }
