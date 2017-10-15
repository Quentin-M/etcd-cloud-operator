# etcd-cloud-operator

Inspired by the [etcd-operator] designed for Kubernetes, the etcd-cloud-operator
manages etcd clusters deployed on cloud providers and helps human operators keep
the data store running safely, even in the event of availability-zone wide
failures.

Used in place of the etcd binary and with minimal configuration, the operator
handles the configuration and lifecycle of etcd, based on data gathered from
the cloud provider and the status of the etcd cluster itself.

The operator makes the assumption that it can trust the cloud provider's
auto-scaling group feature to provide accurate information regarding the number
of launched instances, and to automatically kill/re-provision crashed ones (e.g.
when an rack switch went down or simply when the service health check has been
failing for an extended amount of time).

## Features

- *Failure recovery*: Upon failure of a minority of the etcd members, the
 operator will automatically attempt to restart (rejoin if necessary) the member
 it manages, thus recovering from the failure.
- *Disaster recovery*: In the event of a failure of the majority of the members,
 resulting in the loss of quorum, the operator may try (if enabled) to seed a
 new cluster from a backup, once the expected amount of instances are present
 and the failed etcd cluster has been shot in the head (after forced backup of
 its remaining healthy members).
- *Snapshots*: The operator realizes backups of each etcd member periodically,
 to enable automated disaster recovery or manual recovery in case of force
 majeure.
- *Resize*: By abstracting the cluster management, resizing the cluster becomes
 straightforward as the underlying auto-scaling group can simply be scaled as
 desired.

The operator and etcd cluster can be easily configured using a [YAML file]. The
configuration notably includes clients/peers TLS encryption/authentication, with
the ability to automatically generate self-signed certificates if encryption
is desired but authentication is not.

## How to try it?

Running a managed etcd cluster using the operator is simply a matter of running
the operator binary in a supported auto-scaling group (as of today, AWS only).

A Terraform [module] is available to easily try the operator out or integrate it
into your infrastructure.

## Additional areas of interest

- Exposing Prometheus data about the cluster's health and resource usage,
including the availability zones spread where etcd is deployed.
- Document use-cases, user-stories and statistics regarding failures.
- Adding support for major cloud-providers, such as Azure and GKE.

[etcd-operator]: https://github.com/coreos/etcd-operator
[YAML file]: config.example.yaml
[module]: terraform/platforms/aws