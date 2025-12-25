use std::collections::HashMap;

use apache_avro::AvroSchema;
use apache_avro::to_value;
use criterion::{Criterion, criterion_group, criterion_main};

mod values;

use avroarrow;

fn benchmark_append_record(c: &mut Criterion) {
    c.bench_function("avroarrow_append_record", |b| {
        let registry = HashMap::<apache_avro::schema::Name, apache_avro::Schema>::default();
        let person_avro_schema = values::Person::get_schema();
        let mut builder = avroarrow::create_builder(&person_avro_schema, 1024, &registry).unwrap();

        b.iter_batched_ref(
            || {
                let p = values::Person::new();
                to_value(&p).unwrap()
            },
            |record| {
                avroarrow::append_record(&mut builder, &person_avro_schema, record).unwrap();
            },
            criterion::BatchSize::NumIterations(100000),
        );
    });
}

criterion_group!(benches, benchmark_append_record);
criterion_main!(benches);
