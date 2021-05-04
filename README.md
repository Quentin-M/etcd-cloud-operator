# etcd-cloud-operator

Maintained by a former CoreOS engineer and inspired from the [etcd-operator]
designed for Kubernetes, the etcd-cloud-operator automatically bootstraps,
monitors, snapshots and recovers etcd clusters on cloud providers.

Used in place of the etcd binary and with minimal configuration, the operator
handles the configuration and lifecycle of etcd, based on data gathered from
the cloud provider and the status of the etcd cluster itself.

In other words, the operator operator is meant to help human operators sleep
at night, while their mysterious etcd data store keeps running safely, even
in the event of process, instance, network, or even availability-zone wide
failures.

## Features

-   _Resize_: By abstracting cluster management, resizing the cluster becomes
    straightforward as the underlying auto-scaling group can simply be scaled as
    desired.

-   _Snapshots_: Periodically, snapshots of the entire key-value space are
    captured, from each of the etcd members and uploaded to an encrypted external
    storage, allowing the etcd (or human) operator to restore the store at a later
    time, in any etcd cluster or instance.

-   _Failure recovery_: Upon failure of a minority of the etcd members, the
    managed members automatically restarts and rejoins the cluster without
    breaking quorum or causing visible downtime - First by simply trying to rejoin
    with their existing data set, otherwise trying to join as a new member with a
    clean state, or by replacing the entire instance if necessary.

-   _Disaster recovery_: In the event of a quorum loss, consequence of the
    simultaneous failure of a majority of the members, the operator coordinates
    to snapshot any live members and cleanly stop then, before seeding a new cluster
    from the latest data revision available once the expected amount of instances
    are ready to start again.

-   _ACL support_: A user can configure the ACL of etcd by providing an **init-acl** config
    in the config file. See [init-acl.md](./docs/init-acl.md) for more information.

-   _JWT auth token support_: JWT auth token can be enabled by specifying the
    `jwt-auth-token-config` in the config file, similar to the etcd [-auth-token](https://etcd.io/docs/v3.3/op-guide/configuration/#--auth-token)
    flag.
    The JWT auth token is **HIGHLY** recommended for [production deployment](https://etcd.io/docs/v3.2/learning/auth_design/#two-types-of-tokens-simple-and-jwt),
    especially when the **init-acl** config is also enabled, the JWT auth token can help
    avoid the potential [invalid auth token issue](https://github.com/etcd-io/etcd/issues/9629).

The operator and etcd cluster can be easily configured using a [YAML file]. The
configuration notably includes clients/peers TLS encryption/authentication, with
the ability to automatically generate self-signed certificates if encryption
is desired but authentication is not.

A changelog is maintained at [CHANGELOG.md](CHANGELOG.md).

## How to try it?

Running a managed etcd cluster using the operator is simply a matter of running
the operator binary in a supported auto-scaling group (as of today, AWS and Kubernetes only).

-   _AWS_: You will need to provide IAM credentials with the following capabilities
    in the container's environment, scoped to the appropriate instances:
    "ec2:DescribeInstances"
    "autoscaling:DescribeAutoScalingGroups"
    "autoscaling:DescribeAutoScalingInstances"

-   Kubernetes: You can run the etcd-cloud-operator in a statefulset, but you will need to provide a
    few environment variables. See the [Readme](docs/kubernetes/README.md) for the `sts` provider.
    The easiest way to get to going is to use the included `helm` [chart](chart/etcd-cloud-operator).

A Terraform [module] is available to easily bring up production-grade etcd clusters
managed by the the operator out, and integrate them into your infrastructure.

[etcd-operator]: https://github.com/coreos/etcd-operator
[yaml file]: config.example.yaml
[module]: terraform/platforms/aws
