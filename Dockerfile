FROM golang:1.11.2-alpine AS build-env

WORKDIR /go/src/github.com/quentin-m/etcd-cloud-operator

# Install & Cache dependencies
RUN apk add --no-cache git curl

RUN apk add --update openssl && \
    wget https://github.com/coreos/etcd/releases/download/v3.3.3/etcd-v3.3.3-linux-amd64.tar.gz -O /tmp/etcd.tar.gz && \
    mkdir /etcd && \
    tar xzvf /tmp/etcd.tar.gz -C /etcd --strip-components=1 && \
    rm /tmp/etcd.tar.gz && \
    go get -u github.com/golang/dep/...

# Install ECO
COPY . .
RUN dep ensure
RUN go install github.com/quentin-m/etcd-cloud-operator/cmd/operator
RUN go install github.com/quentin-m/etcd-cloud-operator/cmd/tester

# Copy ECO and etcdctl into an Alpine Linux container image.
FROM alpine

RUN apk add --no-cache ca-certificates
COPY --from=build-env /go/bin/operator /operator
COPY --from=build-env /go/bin/tester /tester
COPY --from=build-env /etcd/etcdctl /usr/local/bin/etcdctl


ENTRYPOINT ["/operator"]
CMD ["-config", "/etc/eco/eco.yaml"]
