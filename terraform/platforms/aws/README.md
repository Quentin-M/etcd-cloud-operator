# etcd-cloud-operator (AWS)

This Terraform module runs the etcd-cloud-operator on private instances of an
AWS ASG inside a VPC - and exposes it through an ELB (that can be either public
or internal).

## Run the operator standalone

If you do not have an existing VPC and subnets across availability zones, you
may create these using the [extra/aws_network] module. The VPC and subnets IDs
will be displayed as a result.

```
terraform apply terraform/extra/aws_network
```

A Terraform configuration file (terraform.tfvars) should then be created. Note
that all available configuration knobs are not exposed.

```
# Name of the deployment.
name = "eco-example"
# Number of etcd members (must be odd).
size = "3"
# Type of the EC2 instances to launch.
instance_type = "t2.small"
# Size of the disk associated to the EC2 instances (in GB).
instance_disk_size = "30"
# Name of the SSH key to use (must be present on EC2).
instance_ssh_key_name = "qmachu-local"

# Defines whether public IPs should be assigned to the EC2 instances (mainly depends if public or private subnets are used).
associate_public_ips = "true"
# List of the subnet IDs to place the EC2 instances in (should span across AZs for availability).
subnets_ids = ["subnet-f438f793", "subnet-d4bea38c"]
# ID of the VPC where the subnets are defined.
vpc_id = "vpc-19f019"
# Defines whether the load balancer for etcd will be internet facing or internal.
load_balancer_internal = "false"
# List of the security group IDs to apply to the load balancer (ingress TCP 2379) (if empty, defaults to open to all).
load_balancer_security_group_ids = []

# Container image of ECO to use.
eco_image = "qmachu/etcd-cloud-operator:latest"
# Defines whether etcd should expect TLS clients connections.
eco_enable_tls = "true"
# Defines whether etcd should expect client certificates for client connections.
eco_require_client_certs = "false"
# Defines whether automatic disaster recovery on ECO should be enabled.
eco_auto_disaster_recovery = "true"
# Defines the interval between consecutive etcd snapshots (e.g. 30m).
eco_snapshot_interval = "30m"
# Defines the lifespan of each etcd snapshot (e.g. 24h).
eco_snapshot_ttl = "24h"
```

Finally, let Terraform configure and create the infrastructure:

```
terraform apply terraform/platforms/aws
```

After a few minutes, the etcd cluster will be available behind the endpoint
displayed. If client certificates authentication was enabled, they will be
displayed as well.

## Integrate the operator into your own project

Just like any other Terraform module, it can be used integrated into remote
projects. Simply add the following block in your code, with the configuration
relevant to your infrastructure:

```
module "eco" {
  source = "github.com/Quentin-M/etcd-cloud-operator//terraform/platforms/aws"

  name = "eco-example"
  max_size = "7"
  min_size = "3"
  desired_capacity = "3"
  min_elb_capacity = "3"

  instance_type         = "t2.small"
  instance_disk_size    = "30"
  instance_ssh_key_name = "qmachu-local"

  associate_public_ip_address      = "true"
  subnets_ids                      = ["subnet-f438f793", "subnet-d4bea38c"]
  vpc_id                           = "vpc-19f019"
  load_balancer_internal           = "false"
  load_balancer_security_group_ids = []

  eco_image                  = "qmachu/etcd-cloud-operator:latest"
  eco_enable_tls             = "true"
  eco_require_client_certs   = "false"
  eco_auto_disaster_recovery = "true"
  eco_snapshot_interval      = "30m"
  eco_snapshot_ttl           = "24h"
}
```

The following outputs will be available for use:
- `etcd_address`
- `ca`, `clients_cert`, `clients_key` (client certificate authentication)
