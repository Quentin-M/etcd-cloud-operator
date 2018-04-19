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

- *Resize*: By abstracting cluster management, resizing the cluster becomes
 straightforward as the underlying auto-scaling group can simply be scaled as
 desired.
 
- *Snapshots*: Periodically, snapshots of the entire key-value space are 
 captured, from each of the etcd members and uploaded to an encrypted external
 storage, allowing the etcd (or human) operator to restore the store at a later
 time, in any etcd cluster or instance.
  
- *Failure recovery*: Upon failure of a minority of the etcd members, the
 managed members automatically restarts and rejoins the cluster without
 breaking quorum or causing visible downtime - First by simply trying to rejoin
 with their existing data set, otherwise trying to join as a new member with a
 clean state, or by replacing the entire instance if necessary.
 
- *Disaster recovery*: In the event of a quorum loss, consequence of the 
 simultaneous failure of a majority of the members, the operator coordinates
 to snapshot any live members and cleanly stop then, before seeding a new cluster
 from the latest data revision available once the expected amount of instances
 are ready to start again.

The operator and etcd cluster can be easily configured using a [YAML file]. The
configuration notably includes clients/peers TLS encryption/authentication, with
the ability to automatically generate self-signed certificates if encryption
is desired but authentication is not.

## How to try it?

Running a managed etcd cluster using the operator is simply a matter of running
the operator binary in a supported auto-scaling group (as of today, AWS only).

- *AWS*: You will need to provide IAM credentials with the capability
 DescribeAutoScalingInstances in the container's environment.

A Terraform [module] is available to easily bring up production-grade etcd clusters
managed by the the operator out, and integrate them into your infrastructure.

[etcd-operator]: https://github.com/coreos/etcd-operator
[YAML file]: config.example.yaml
[module]: terraform/platforms/aws