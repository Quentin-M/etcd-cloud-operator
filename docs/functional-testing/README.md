## Functional Testing

The functional test suite verifies that the an etcd cluster, managed by the 
etcd-cloud-operator, is able to cope with severe failures injected while the cluster
is under high pressure. This is similar to the upstream's [functional test suite].

There are obviously smarter ways to automate/run this test suite, but its development
time is limited, therefore focus was on making this happen quickly.

### Running the tests.

- Make sure the ECO deployment has a sufficient backend quota (e.g. `8589934592`).
- Open port 22 on the security group for ECO instances

- Edit the present config.yaml to match the ECO deployment
- Create an Ubuntu EC2 instance in the same VPC as the ECO deployment, with ports 22/3000 opened
- Open four shells and execute the following commands:

```
fswatch -o ./ | while read num; do rsync -avz ./ ubuntu@<ubuntu instance's address>
```

```
ssh -A ubuntu@<ubuntu instance's address>
sudo -E su
apt update && apt install docker.io
curl -L https://github.com/docker/compose/releases/download/1.18.0/docker-compose-`uname -s`-`uname -m` -o /usr/bin/docker-compose && chmod +x /usr/bin/docker-compose
cd /home/ubuntu/docs/testing && docker-compose up
```

```
ssh -A ubuntu@<ubuntu instance's address>
sudo -E su
docker exec -it $(docker ps|grep tester|awk '{print $1}') bash

cd /go/src/github.com/quentin-m/etcd-cloud-operator/docs/testing/
go install -v github.com/quentin-m/etcd-cloud-operator/cmd/tester && tester -config=config.yaml -log-level=debug
```

```
ssh -A ubuntu@<ubuntu instance's address>
watch e --endpoints=<1rd instance's ip>:2379,<2nd instance's ip>:2379,<3rd instance's ip>:2379 --dial-timeout=1s --command-timeout=1s endpoint status -w table
```

- Login to `http://<ubuntu instance's address>:3000` with `admin:password`

[functional test suite]: https://github.com/coreos/etcd/tree/master/tools/functional-tester