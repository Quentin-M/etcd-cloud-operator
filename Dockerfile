FROM golang:1.9-alpine AS build-env
WORKDIR /go/src/github.com/quentin-m/etcd-cloud-operator
COPY . .
RUN go-wrapper install github.com/quentin-m/etcd-cloud-operator/cmd/operator

FROM alpine
COPY --from=build-env /go/bin/operator /operator
RUN apk add --no-cache ca-certificates
RUN apk add --no-cache --repository http://dl-3.alpinelinux.org/alpine/edge/testing/ etcd-ctl
ENTRYPOINT ["/operator"]
CMD ["-config /etc/eco/eco.yaml"]
