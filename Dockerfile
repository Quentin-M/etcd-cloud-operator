#Min version required
#See: https://github.com/golang/go/issues/29278#issuecomment-447537558
FROM golang:1.25-alpine3.22 AS build-env

WORKDIR /go/src/github.com/quentin-m/etcd-cloud-operator

# Install & cache dependencies
RUN apk add --no-cache git curl gcc musl-dev ca-certificates openssl wget
RUN update-ca-certificates

# Force the go compiler to use modules
ENV GO111MODULE=on

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.* .
RUN go mod download

FROM build-env AS builder
COPY . .
RUN go install github.com/quentin-m/etcd-cloud-operator/cmd/operator
RUN go install github.com/quentin-m/etcd-cloud-operator/cmd/tester

# Copy ECO and etcdctl into an Alpine Linux container image.
FROM alpine

RUN apk add --no-cache ca-certificates docker-cli
RUN update-ca-certificates
COPY --from=builder /go/bin/operator /operator
COPY --from=builder /go/bin/tester /tester
COPY --link --from=gcr.io/etcd-development/etcd:v3.6.4 /usr/local/bin/etcdctl /usr/local/bin/etcdctl
COPY --link --from=gcr.io/etcd-development/etcd:v3.6.4 /usr/local/bin/etcdutl /usr/local/bin/etcdutl


ENTRYPOINT ["/operator"]
CMD ["-config", "/etc/eco/eco.yaml"]
