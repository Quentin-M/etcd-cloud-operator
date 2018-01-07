FROM golang:1.9-alpine AS build-env
WORKDIR /go/src/github.com/quentin-m/etcd-cloud-operator
COPY . .
RUN go-wrapper install github.com/quentin-m/etcd-cloud-operator/cmd/operator
RUN apk add --update openssl && \
    wget https://github.com/coreos/etcd/releases/download/v3.3.0-rc.1/etcd-v3.3.0-rc.1-linux-amd64.tar.gz -O /tmp/etcd.tar.gz && \
    mkdir /etcd && \
    tar xzvf /tmp/etcd.tar.gz -C /etcd --strip-components=1 && \
    rm /tmp/etcd.tar.gz

FROM alpine
COPY --from=build-env /go/bin/operator /operator
COPY --from=build-env /etcd/etcdctl /usr/local/bin/etcdctl
RUN apk add --no-cache ca-certificates
ENTRYPOINT ["/operator"]
CMD ["-config /etc/eco/eco.yaml"]
