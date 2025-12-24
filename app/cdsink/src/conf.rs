pub enum ReaderType {
    Kafka,
    StdIn,
}

pub enum FormatterType {
    AvroRecord,
    JsonRecord,
    JsonRecordSchema,
}

pub enum WriterType {
    DeltaLake,
    Iceberg,
}

pub struct Conf {}
