set dotenv-path := '.env'

export GOOSE := 'migrate'

mod connect 'env/connect/mod.just'
mod migrate 'env/goose/mod.just'

psql:
    @docker compose exec -e PGUSER=$POSTGRES_USER -e PGPASSWORD=$POSTGRES_PASSWORD db psql

run *FLAGS:
    @deltasink/target/debug/deltasink {{FLAGS}}
