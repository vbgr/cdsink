ARG ALPINE_VERSION=3.22
ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as build
WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" ./cmd/randb

FROM alpine:${ALPINE_VERSION}
COPY --from=build /src/randb /usr/local/bin
RUN adduser -D -h /app app && apk add jq
USER app
ENTRYPOINT [ "randb" ]
