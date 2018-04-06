FROM golang:1.9-alpine AS build-env

WORKDIR /go/src/github.com/quentin-m/etcd-cloud-operator

# Install & Cache dependencies
RUN apk add --no-cache git jq && \
    go get github.com/Masterminds/glide && \
    go get github.com/creack/yaml2json

RUN apk add --update openssl && \
    wget https://github.com/coreos/etcd/releases/download/v3.3.3/etcd-v3.3.3-linux-amd64.tar.gz -O /tmp/etcd.tar.gz && \
    mkdir /etcd && \
    tar xzvf /tmp/etcd.tar.gz -C /etcd --strip-components=1 && \
    rm /tmp/etcd.tar.gz

ADD glide.* ./
RUN glide install --strip-vendor && yaml2json < glide.lock | \
      jq -r -c '.imports[], .testImports[] | {name: .name, subpackages: (.subpackages + [""])}' | \
      jq -r -c '.name as $name | .subpackages[] | [$name, .] | join("/")' | sed 's|/$||' | \
      while read pkg; do \
        echo "${pkg}..."; \
        go install ./vendor/${pkg} 2> /dev/null; \
      done

# Fetch etcdctl
COPY . .
RUN go-wrapper install github.com/quentin-m/etcd-cloud-operator/cmd/operator
RUN go-wrapper install github.com/quentin-m/etcd-cloud-operator/cmd/tester

# Install ECO
COPY . .
RUN go-wrapper install github.com/quentin-m/etcd-cloud-operator/cmd/operator
RUN go-wrapper install github.com/quentin-m/etcd-cloud-operator/cmd/tester

# Copy ECO and etcdctl into an Alpine Linux container image.
FROM alpine

COPY --from=build-env /go/bin/operator /operator
COPY --from=build-env /go/bin/tester /tester
COPY --from=build-env /etcd/etcdctl /usr/local/bin/etcdctl

RUN apk add --no-cache ca-certificates

ENTRYPOINT ["/operator"]
CMD ["-config /etc/eco/eco.yaml"]
