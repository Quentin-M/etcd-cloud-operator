module github.com/quentin-m/etcd-cloud-operator

go 1.13

require (
	github.com/aws/aws-sdk-go v1.29.19
	github.com/coreos/pkg v0.0.0-20160727233714-3ac0863d7acf
	github.com/prometheus/client_golang v1.0.0
	github.com/sirupsen/logrus v1.4.2
	go.etcd.io/etcd v0.0.0-20200824191128-ae9734ed278b
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	google.golang.org/grpc v1.26.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
)
