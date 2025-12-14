# Testing approach

The testing approach defined in this document targets the following goals:

1. [Goal 1] Proof that the application produces 100% correct data and its
   metadata.
2. [Goal 2] The application still works correct if experiences outages.
3. [Goal 3] The application still works correct if third party components it
   deponds on are down (partially or fully).
4. [Goal 4] Proof that application can scale dynamically and still work correct.
5. [Goal 5] Provide reproducable local environment that allows to proof all of
   the above statements.

## Integration End to End tests

Only full end to end tests can address the Goal 1. Therefore they are
prioritized at the initial phase over unit tests. It does not mean unit tests
should be ignored completely but they should be considered as a method that
helps during development rather than proof of the correctness of the entire
application.

### Required environment

1. Docker Compose. Standard de facto for providing easy isolated local
   environment.
2. Kafka deployed as 3 brokers. Distributed setup is required to proof that
   application still work correct during rebalance.
3. Minio deployed as 4 nodes as S3 storage. Distributed setup is required to
   prof that the application still works correct if S3 refuses to ingest the
   data.
4. PostgreSQL, single instance as source of CDC evnets.
5. `randb` as database random writes tool.
6. Kafka Connect. Even though Debezium can be executed as dedicated server
   instance more common setup is Kafka Connect. This option also provides good
   visualization via UI for Apache Kafka.
7. Debezium Connector as CDC events streamer.
8. Apicurio schema registry. Reuired to test Avro format.
9. Lakekeeper. Required to test optional but important Data Catalog integration.

### Test cases

#### [T-001] Core CDC Functionality and Atomic Upserts

**Objective**

Proof that cdsink correctly handles a mixed transactional workload
(INSERT, UPDATE, DELETE) and maintains 100% data fidelity in the Lakehouse,
including correct metadata registration.

**Steps**

1. Setup environment. Apply all database migrations.
2. Start cdsink consuming Avro records and streaming into Delta Lake.
3. Run randb tool to emulate a diverse, high-volume mix of INSERT, UPDATE, and
   DELETE operations.
4. Wait 5 minutes for substantial traffic.
5. Stop randb and wait until cdsink logs confirm all Kafka messages are
   processed.
6. Validate Data Completeness: Export final, sorted records from PostgreSQL and
   the Delta Lake table. Assert that the final record count and the
   checksum/digest of the sorted data are identical.
7. Validate Metadata: Check the Lakekeeper Data Catalog to confirm the table is
   registered and the schema matches the final expected schema.

#### [T-002] Automatic Schema Evolution and Data Integrity

**Objective**

Proof that cdsink correctly handles schema drift (adding columns) mid-stream
without data loss, corruption, or manual intervention.

**Steps**

1. Setup environment. Apply initial database migration.
2. Start cdsink.
3. Run randb tool to generate initial data for 30 seconds.
4. Apply next DB migration (adding one or more nullable columns).
5. Continue running randb to generate data reflecting the new schema.
6. Repeat steps 4-5 until all schema migrations are applied.
7. Stop all traffic and wait for processing completion.
8. Validate Schema: Assert the Delta Lake table metadata and the Lakekeeper
   catalog show the final, correct schema (all columns present).
9. Validate Data Completeness: Assert the final record count and checksum of the
   source (PostgreSQL) and target (Delta Lake) data are identical, verifying no
   data loss occurred during schema transitions.

[T-003] Kafka Chaos: Resilience to Broker Outage and Rebalance

**Objective**

Proof that cdsink maintains correct transactional state, ordering, and avoids
data duplication/loss when the Kafka message bus experiences an unplanned broker
failure.

**Steps**

1. Start continuous, high-volume randb traffic.
2. After 1 minute, terminate one of the three Kafka broker instances (simulating
   an unplanned outage).
3. Monitor cdsink logs to confirm it detects the failure, re-establishes its
   consumer connection, and re-reads partitions correctly.
4. Continue traffic for 2 minutes during the recovery phase.
5. Stop all traffic.
6. Assert Data Correctness (checksum/count), confirming no data loss or
   corruption occurred during the rebalance event.

#### [T-004] Minio Chaos: Idempotency and Commit Retry

**Objective**

Proof that cdsink can successfully handle intermittent network issues or write
failures to object storage without partial commits or data loss, verifying
idempotency.

**Steps**

1. Start continuous randb traffic.
2. Use Docker/network tools to block all network traffic to the Minio S3
   endpoints for 45 seconds (simulating a brownout during a write or commit
   phase).
3. Verify cdsink successfully buffers the micro-batch, or correctly attempts to
   retry the write and the transactional commit.
4. Restore network access.
5. Wait for the backlog to clear.
6. Assert Data Correctness, verifying atomicity (the write and commit succeeded
   cleanly on retry, or failed and were retried later) and that there are no
   duplicate records in the final Delta Lake table.


#### [T-005] Dynamic Scaling Up and Down

**Objective**

Proof that cdsink maintains correct state and resumes processing cleanly when
the number of running application instances (and consumer group members)
changes.

1. Start cdsink with one replica. Start high-volume randb traffic. 
2. After 2 minutes, scale up cdsink to three replicas (simulating dynamic
   scaling or blue/green deployment). 
3. Verify that the Kafka consumer group rebalances correctly, and all three
   instances begin consuming partitions. 
4. After 2 minutes, scale down cdsink back to one replica. 
5. Verify the remaining instance resumes consuming all partitions correctly. 
6. Stop all traffic. 
7. Assert Data Correctness (checksum/count), confirming no data was dropped or
   duplicated during the scale events.

#### [T-006] Sustained Performance and Resource Stability

**Objective**

Proof that cdsink maintains a stable resource profile and low latency under
sustained, high-volume transactional load over an extended period.

**Steps**

1. Generate peak sustained traffic (e.g., 5,000 records/sec) using randb for 8
   hours.
2. Monitor resource usage: CPU, Memory, and Network I/O of the cdsink container.
3. Verify that the Kafka consumer lag remains low and stable.
4. Assert the final data is correct and that resource usage did not leak or
   spiral out of control (i.e., verify performance NFRs).


#### [T-007] Dead Letter Queue Functionality

Proof that cdsink correctly identifies malformed records, diverts them to the
DLQ, and continues processing valid records without interruption or data loss.

**Steps**

1. Configure cdsink with a specific DLQ target (e.g., Minio path in JSON
   format). 
2. Start cdsink and continuous randb traffic (valid data). 
3. Manually insert a sequence of intentionally malformed Kafka records (e.g.,
   records with null values in non-nullable fields, or corrupted Avro
   serialization). 
4. Stop all traffic and wait for processing completion. 
5. Assert Validity: Verify the main Delta Lake table contains only the data
   generated by randb (correct records). 
6. Assert DLQ: Verify the DLQ Minio path contains only the sequence of malformed
   records, and they are written in the correct format (JSON/CSV).

#### [T-008] In-Process SQL Transformation Correctness

**Objective**

Proof that optional SQL transformations (filtering, column renaming, simple
aggregation) execute correctly in-process without degrading data correctness or
transactional integrity.

1. Configure cdsink to use a non-trivial SQL transformation (e.g., SELECT
user_id, CAST(value * 1.1 AS INT) AS adjusted_value FROM source WHERE status =
'active'). 
2. Start cdsink and run randb to generate a known set of input data (active and
   inactive users). 
3. Wait for processing completion. 
4. Validate Transformation Logic: Export the final Delta Lake table. Assert
   that: a) The output schema reflects the transformation (new column
   names/types). b) Only records meeting the WHERE clause condition were
   processed. c) The calculated column values (e.g., adjusted_value) are
   mathematically correct.
