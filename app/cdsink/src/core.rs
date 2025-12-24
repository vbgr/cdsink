//! # CDSync Core Abstractions
//!
//! This module serves as the "Domain Model" for the CDSync data pipeline. It defines
//! the shared types, error hierarchies, and trait interfaces that allow disparate
//! data sources (Kafka, S3, etc.) to communicate with analytical targets (Delta Lake, Parquet).
//!
//! ### Architecture Overview
//! The pipeline follows a **Source -> Transform -> Sink** pattern:
//! 1. **Readers** pull raw data and encapsulate them into `SourceBatch` units.
//! 2. **Converters** bridge the gap between raw bytes (Avro/JSON) and Arrow memory.
//! 3. **Writers** handle the durability of the `RecordBatch` to the final destination.
//!
//! ### Concurrency and Safety
//! Traits are designed to be `async` and thread-safe, enabling high-performance
//! multi-table ingestion while maintaining strict sequencing for data integrity.

#![allow(dead_code)]

use std::hash::{DefaultHasher, Hash, Hasher};
use std::io;

use async_trait::async_trait;
use datafusion::arrow::array::ArrayBuilder;
use datafusion::arrow::datatypes::{Field, SchemaRef};
use datafusion::arrow::record_batch::RecordBatch;
use thiserror::Error;
use tracing::error;

/// Represents the structural differences between two Arrow Schemas.
///
/// This struct is used by the `Writer` to determine which metadata actions
/// (e.g., ADD COLUMN, ALTER COLUMN) are required to synchronize the
/// target table with the incoming batch.
#[derive(Debug, Clone, Default)]
pub struct SchemaDiff {
    /// New fields found in the source that do not exist in the target table.
    /// In Lakehouse systems, these typically trigger an `ADD COLUMN` operation.
    pub created: Vec<Field>,

    /// Fields that exist in both schemas but have mismatched types or nullability.
    /// The tuple contains `(old_field, new_field)`.
    ///
    /// ### Warning:
    /// Check for 'Type Promotion' safety before applying (e.g., Int32 -> Int64 is
    /// usually safe, but String -> Int is a breaking change).
    pub updated: Vec<(Field, Field)>,

    /// Fields present in the target table but missing from the source batch.
    ///
    /// ### Note:
    /// In many append-only pipelines, these are ignored to avoid data loss,
    /// as "deleting" a column in the target is a destructive operation.
    pub deleted: Vec<Field>,
}

/// Errors occurring during the translation of raw source data into Arrow arrays.
#[derive(Error, Debug)]
pub enum ConversionError {
    /// The input data does not match the expected format or schema.
    #[error("Malformed payload: {0}")]
    InvalidFormat(String),
    /// The converter encountered a value that cannot be represented in the target Arrow type.
    #[error("Value conversion failed: {0}")]
    TypeMismatch(String),
}

/// Errors specific to data ingestion and source-system interactions.
#[derive(Error, Debug)]
pub enum ReaderError {
    /// Permission issues when accessing the source (e.g., Kafka ACLs or S3 Bucket policies).
    #[error("Access denied: {0}")]
    Access(String),
    /// Underlying hardware or network failures.
    #[error("IO error: {0}")]
    Io(#[from] io::Error),
    /// Data was retrieved but failed integrity checks (e.g., checksum failure).
    #[error("Corrupted data: {0}")]
    Corrupted(String),
    /// The source failed to respond within the configured deadline.
    #[error("Timeout error: {0}")]
    Timeout(String),
    /// A failure occurred during the immediate conversion of fetched records.
    #[error("Conversion error: {0}")]
    ConversionError(ConversionError),
}

/// Errors occurring during data persistence to the target lakehouse or storage.
#[derive(Error, Debug)]
pub enum WriterError {
    /// Permission issues when writing to the target storage.
    #[error("Access denied: {0}")]
    Access(String),
    /// Underlying hardware or network failures during the write phase.
    #[error("IO error: {0}")]
    Io(#[from] io::Error),
    /// Target system failed to acknowledge the write within the deadline.
    #[error("Timeout error: {0}")]
    Timeout(String),
    /// The data's schema does not match the existing table schema at the target.
    #[error("Schema mismatch: {0}")]
    Schema(String),
    /// Physical storage limits reached (critical for local buffers or disk-based sinks).
    #[error("Storage full")]
    NoSpace,
}

/// The top-level error type for the application, facilitating unified error handling
/// in the main supervisor loop.
#[derive(Error, Debug)]
pub enum AppError {
    #[error("Reader error: {0}")]
    Reader(#[from] ReaderError),
    #[error("Converter error: {0}")]
    Converter(#[from] ConversionError),
    #[error("Writer error: {0}")]
    Writer(#[from] WriterError),
    #[error("To many errors")]
    ToManyErrors,
}

/// Represents the partitioning key of the data in its native or logical format.
///
/// Using an enum avoids unnecessary string allocations and parsing overhead
/// for numerical keys while remaining flexible for string-based keys.
pub enum Partition {
    /// A string-based partition identifier (e.g., "region=us-east-1").
    Named(String),
    /// A numerical partition identifier (e.g., a Unix timestamp). Efficient for branch prediction.
    Numeric(i64),
}

/// Metadata that describes the origin and destination context of a data batch.
pub struct SourceMetadata {
    /// The logical name of the target entity (e.g., the Delta Lake table name).
    pub table: String,
    /// The logical partition where this data belongs.
    pub partition: Partition,
    /// The specific point in the source system that this batch covers.
    pub position: SourcePosition,
}

impl SourceMetadata {
    pub fn hash(&self) -> u64 {
        let mut hasher = DefaultHasher::new();
        self.table.hash(&mut hasher);
        hasher.finish()
    }
}

/// Defines the specific location or cursor within a data source.
pub enum SourcePosition {
    /// A unique sequence number or offset in a stream (e.g., Kafka).
    Offset(i64),
    /// The unique identifier or path for a discrete object (e.g., S3 key).
    File(String),
}

/// A self-contained unit of work containing both data and its descriptive metadata.
pub struct SourceBatch {
    /// Contextual information used for routing and acknowledgment.
    pub metadata: SourceMetadata,
    /// The actual columnar data payload.
    pub records: RecordBatch,
}

/// The core interface for ingestion components.
#[async_trait]
pub trait Reader {
    /// Retrieves the next available batch of data from the source.
    async fn fetch(&self) -> Result<Option<SourceBatch>, ReaderError>;

    /// Finalizes the processing of a batch in the source system (e.g., Commits Kafka offsets).
    async fn commit(&self, metadata: &[SourceMetadata]) -> Result<(), ReaderError>;
}

/// The execution engine for a specific target table.
///
/// A `Writer` encapsulates the logic required to communicate with a
/// specific storage backend (e.g., Delta Lake, Iceberg). It manages
/// both the data plane (insert/upsert) and the control plane (schema evolution).
#[async_trait]
pub trait Writer: Send + Sync {
    /// Retrieves the current physical schema of the target table.
    ///
    /// This should perform a metadata refresh (e.g., checking S3 for
    /// the latest Delta Log or Iceberg Metadata JSON) to ensure the
    /// returned schema accounts for changes made by other processes.
    async fn schema(&self) -> Result<SchemaRef, WriterError>;

    /// Applies structural changes to the table.
    ///
    /// This is typically a metadata-only transaction. Implementation
    /// should handle optimistic concurrency—if another process evolves
    /// the schema simultaneously, this should return a retryable
    /// conflict error.
    async fn alter(&self, diff: &SchemaDiff) -> Result<(), WriterError>;

    /// Appends a batch of records to the table.
    ///
    /// This assumes the batch schema is already compatible with the
    /// table. If a schema mismatch is detected at the storage layer,
    /// it should return an error indicating a refresh is required.
    async fn insert(&self, batch: &SourceBatch) -> Result<(), WriterError>;

    /// Merges a batch of records into the table based on primary keys.
    ///
    /// Performs an idempotent "upsert." This operation is more complex
    /// than `insert` as it requires matching existing records to
    /// determine whether to update or create.
    async fn upsert(&self, batch: &SourceBatch) -> Result<(), WriterError>;
}

/// A handle to a storage-layer table resource.
///
/// The `Table` trait is responsible for bootstrapping the connection
/// to the underlying storage and producing a `Writer`. It acts as
/// the entry point for the pipeline to interact with a specific destination.
#[async_trait]
pub trait Table {
    /// Instantiates a `Writer` for this table.
    ///
    /// This involves loading table metadata, initializing connection
    /// pools, and verifying that the target path exists and is
    /// accessible.
    async fn get_writer(&self) -> Result<Box<dyn Writer>, WriterError>;
}

/// Bridges raw binary data to Arrow memory structures.
///
/// This trait allows the ingestion logic to remain agnostic of the data format
/// (Avro, JSON, Protobuf).
pub trait RecordConverter {
    /// Extracts or computes a unique schema identifier from the raw payload.
    ///
    /// This is used to ensure "Batch Homogeneity." If a schema change is detected
    /// mid-stream, the current batch must be flushed before processing records
    /// with the new `schema_id`.
    fn get_schema_id(&self, payload: &[u8]) -> i32;

    /// Translates raw bytes into Arrow array data.
    ///
    /// Implementations should append the parsed data directly into the provided
    /// `ArrayBuilder`. This minimizes intermediate allocations.
    fn convert(
        &self,
        payload: &[u8],
        builder: &mut dyn ArrayBuilder,
    ) -> Result<(), ConversionError>;
}

/// Represents the data payload captured at the point of failure.
///
/// This enum allows the DLQ to be polymorphic: it can store raw bytes if
/// the failure happened before conversion, or structured batches if the
/// failure happened during the write to the target.
pub enum DlqPayload {
    /// Captured when the `RecordConverter` fails.
    /// Contains the original bytes (e.g., Avro, JSON, or Protobuf) to allow
    /// for later debugging or manual re-ingestion once the schema/logic is fixed.
    Raw(Vec<u8>),

    /// Captured when the `Writer` fails.
    /// Contains the already-processed `RecordBatch`. Storing this directly avoids
    /// the CPU cost of re-serializing data that has already been successfully
    /// transformed into columnar format.
    Batch(RecordBatch),
}

/// A comprehensive container for data that could not be processed.
///
/// This structure provides the "Three W's" of error handling:
/// - **Where** did it come from? (`metadata`)
/// - **What** was the data? (`payload`)
/// - **Why** did it fail? (`error_message`)
pub struct DlqMessage {
    /// Contextual origin of the data, including table, partition, and source position.
    pub metadata: SourceMetadata,

    /// The actual data that failed processing.
    pub payload: DlqPayload,

    /// A descriptive error message or stack trace explaining the cause of failure.
    pub error_message: String,
}

/// Interface for persisting failed records to a Dead Letter Queue (DLQ) storage.
///
/// This trait decouples the main pipeline logic from the specific storage
/// medium used for errors (e.g., S3, local disk, or a secondary Kafka topic).
#[async_trait]
pub trait DlqWriter: Send + Sync {
    /// Persists a `DeadLetter` entry to the error storage.
    ///
    /// Implementations are responsible for:
    /// 1. Inspecting the `DeadPayload` (Raw vs Batch).
    /// 2. Routing the data to the appropriate physical location.
    /// 3. Ensuring the `error_message` and `SourceMetadata` are preserved
    ///    alongside the data for later diagnosis.
    ///
    /// ### Performance Note
    /// Since this is called in the failure path, it should be highly resilient.
    /// If the `DQLWriter` itself fails, the pipeline may need to halt to prevent
    /// data loss (silent failures).
    async fn write(&self, dl: Vec<DlqMessage>) -> Result<(), AppError>;
}
