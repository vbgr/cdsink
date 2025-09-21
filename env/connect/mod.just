debezium-avro:
    curl -s -X POST http://localhost:8083/connectors \
        -H 'Content-Type: application/json' \
        -d '\
        { \
            "name": "app-src-avro", \
            "config": { \
                "connector.class": "io.debezium.connector.postgresql.PostgresConnector", \
                "max.tasks": 2, \
                "slot.name": "app_avro", \
                "plugin.name": "pgoutput", \
                "database.hostname": "db", \
                "database.port": "5432", \
                "database.user": "{{env_var('POSTGRES_USER')}}", \
                "database.password": "{{env_var('POSTGRES_PASSWORD')}}", \
                "database.dbname": "app", \
                "database.server.name": "app.avro", \
                "schema.name.adjustment.mode": "avro", \
                "table.include.list": "public.app_.+", \
                "include.schema.comments": true, \
                "column.propagate.source.type": ".*", \
                "topic.prefix": "app.avro", \
                "key.converter": "io.confluent.connect.avro.AvroConverter", \
                "key.converter.schemas.enable": "true", \
                "key.converter.schema.registry.url": "http://s-1:8080/apis/ccompat/v7", \
                "value.converter": "io.confluent.connect.avro.AvroConverter", \
                "value.converter.schemas.enable": "true", \
                "value.converter.schema.registry.url": "http://s-1:8080/apis/ccompat/v7", \
                "transforms": "unwrap", \
                "transforms.unwrap.drop.tombstones": true, \
                "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState", \
                "transforms.unwrap.add.fields": "db,table,op,lsn,source.ts_ms,source.txId" \
            } \
        }'

debezium-json:
    curl -s -X POST http://localhost:8083/connectors \
        -H 'Content-Type: application/json' \
        -d '\
        { \
            "name": "app-src-json", \
            "config": { \
                "connector.class": "io.debezium.connector.postgresql.PostgresConnector", \
                "max.tasks": 2, \
                "slot.name": "app_json", \
                "plugin.name": "pgoutput", \
                "database.hostname": "db", \
                "database.port": "5432", \
                "database.user": "{{env_var('POSTGRES_USER')}}", \
                "database.password": "{{env_var('POSTGRES_PASSWORD')}}", \
                "database.dbname": "app", \
                "database.server.name": "app.json", \
                "schema.name.adjustment.mode": "avro", \
                "table.include.list": "public.app_.+", \
                "include.schema.comments": true, \
                "column.propagate.source.type": ".*", \
                "topic.prefix": "app.json", \
                "key.converter": "org.apache.kafka.connect.json.JsonConverter", \
                "key.converter.schemas.enable": "true", \
                "value.converter": "org.apache.kafka.connect.json.JsonConverter", \
                "value.converter.schemas.enable": "true", \
                "transforms": "unwrap", \
                "transforms.unwrap.drop.tombstones": true, \
                "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState", \
                "transforms.unwrap.add.fields": "db,table,op,lsn,source.ts_ms,source.txId" \
            } \
        }'
