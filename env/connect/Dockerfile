ARG VERSION=7.5.3

FROM alpine:3.20 as debezium
WORKDIR /tmp
RUN mkdir /jars
RUN wget https://repo1.maven.org/maven2/io/debezium/debezium-connector-postgres/2.5.0.Final/debezium-connector-postgres-2.5.0.Final-plugin.tar.gz
RUN tar -xvf debezium-connector-postgres-*.tar.gz
RUN mv -f debezium-connector-postgres/*.jar /jars

FROM alpine:3.20 as aiven
WORKDIR /tmp
RUN mkdir /jars
RUN wget https://github.com/Aiven-Open/transforms-for-apache-kafka-connect/releases/download/v1.5.0/transforms-for-apache-kafka-connect-1.5.0.tar
RUN tar -xvf transforms-for-apache-kafka-connect-*.tar
RUN mv transforms-for-apache-kafka-connect-*/transforms-for-apache-kafka-connect-*.jar /jars

FROM confluentinc/cp-kafka-connect:${VERSION}
COPY --from=debezium /jars /usr/share/java/kafka
COPY --from=aiven /jars /usr/share/java/kafka
RUN echo installing plugins... \
&& confluent-hub install confluentinc/kafka-connect-avro-converter:latest --no-prompt \
&& confluent-hub install jcustenborder/kafka-connect-transform-common:latest --no-prompt
