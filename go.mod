module github.com/quentin-m/etcd-cloud-operator

go 1.13

require (
	github.com/aws/aws-sdk-go v1.29.19
	github.com/coreos/pkg v0.0.0-20160727233714-3ac0863d7acf
	github.com/sirupsen/logrus v1.2.0
	go.etcd.io/bbolt v1.3.3
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200224211402-c65a9e2dd1fd
	google.golang.org/grpc v1.23.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
)
