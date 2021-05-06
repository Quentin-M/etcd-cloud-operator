module github.com/quentin-m/etcd-cloud-operator

go 1.16

replace (
	go.etcd.io/etcd/api/v3 => go.etcd.io/etcd/api/v3 v3.0.0-20210505210143-8af8f6af2700
	go.etcd.io/etcd/client/pkg/v3 => go.etcd.io/etcd/client/pkg/v3 v3.0.0-20210505210143-8af8f6af2700
	go.etcd.io/etcd/client/v3 => go.etcd.io/etcd/client/v3 v3.0.0-20210505210143-8af8f6af2700
	go.etcd.io/etcd/etcdctl/v3 => go.etcd.io/etcd/etcdctl/v3 v3.0.0-20210505210143-8af8f6af2700
	go.etcd.io/etcd/pkg/v3 => go.etcd.io/etcd/pkg/v3 v3.0.0-20210505210143-8af8f6af2700
	go.etcd.io/etcd/server/v3 => go.etcd.io/etcd/server/v3 v3.0.0-20210505210143-8af8f6af2700
	go.etcd.io/etcd/v3 => go.etcd.io/etcd/v3 v3.0.0-20210505210143-8af8f6af2700
	go.etcd.io/etcd/client/v2 => go.etcd.io/etcd/client/v2 v2.0.0-20210505210143-8af8f6af2700
	google.golang.org/grpc => google.golang.org/grpc v1.36.1
)

require (
	github.com/aws/aws-sdk-go v1.38.34
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.3-0.20210316121004-a77ba4df9c27 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/sirupsen/logrus v1.7.0
	github.com/soheilhy/cmux v0.1.5 // indirect
	go.etcd.io/etcd/api/v3 v3.5.0-alpha.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0-alpha.0
	go.etcd.io/etcd/client/v3 v3.5.0-alpha.0
	go.etcd.io/etcd/etcdctl/v3 v3.5.0-alpha.0
	go.etcd.io/etcd/server/v3 v3.5.0-alpha.0
	go.etcd.io/etcd/client/v2 v2.305.0-alpha.0
	go.uber.org/zap v1.16.1-0.20210329175301-c23abee72d19
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.36.1
	gopkg.in/yaml.v2 v2.3.0
	moul.io/zapfilter v1.6.1 // indirect
)
