module github.com/quentin-m/etcd-cloud-operator

go 1.16

require (
	github.com/aws/aws-sdk-go v1.38.34
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.7.0
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	go.etcd.io/etcd/etcdutl/v3 v3.5.0
	go.etcd.io/etcd/server/v3 v3.5.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.38.0
	gopkg.in/yaml.v2 v2.4.0
	moul.io/zapfilter v1.6.1
)
