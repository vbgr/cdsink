ARG ALPINE_VERSION=3.22
ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as build
RUN apk add git
ARG GIT_URL=https://github.com/pressly/goose.git
RUN git clone ${GIT_URL} /src
WORKDIR /src
RUN git checkout ${VERSION}
RUN go mod download
ARG TAGS="no_mysql no_sqlite3 no_ydb no_clickhouse no_mssql no_vertica no_libsql"
RUN go build -ldflags="-s -w" -tags="${TAGS}" -o goose ./cmd/goose

FROM alpine:${ALPINE_VERSION}
COPY --from=build /src/goose /usr/local/bin
RUN adduser -D -h /app app
USER app
ENV GOOSE_DRIVER=postgres
ENTRYPOINT [ "goose" ]
