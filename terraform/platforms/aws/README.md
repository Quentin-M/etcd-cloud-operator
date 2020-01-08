# etcd-cloud-operator (AWS)

This Terraform module runs the etcd-cloud-operator on private instances of an
AWS ASG inside a VPC - and exposes it through an ELB (that can be either public
or internal).

## Run the operator standalone

If you do not have an existing VPC and subnets across availability zones, you
may create these using the [extra/aws_network](../../extra/aws_network) module.
The VPC and subnets IDs will be displayed as a result.

```
cd terraform/extra/aws_network
terraform init .
terraform apply .
```

A Terraform configuration file (terraform/platforms/aws/terraform.tfvars) should
then be created. Note that all available ectd-cloud-operator (ECO) configuration
knobs are not exposed.

```
# Name of the deployment.
name = "eco-example"
# Number of etcd members (must be odd).
size = "3"
# Type of the EC2 instances to launch.
instance_type = "m5.large"
# Size of the disk associated to the EC2 instances (in GB).
instance_disk_size = "30"
# List of SSH public keys that are allowed to login into nodes
instance_ssh_keys = ["ssh-rsa ..."]

# Defines whether public IPs should be assigned to the EC2 instances (mainly depends if public or private subnets are used).
associate_public_ips = true
# List of the subnet IDs to place the EC2 instances in (should span across AZs for availability).
subnets_ids = ["subnet-f438f793", "subnet-d4bea38c"]
# ID of the VPC where the subnets are defined.
vpc_id = "vpc-19f019"
# Defines whether a Route53 record should be created for client connections.
route53_enabled = false
# Optional Route53 Zone ID under which an 'etcd' record should be created for client connections.
route53_zone_id = ""
# Route53 prefix name defines the shortname for the record. Appends to route53_zone_id name for fqdn.
route53_prefix = "" #Default = "etcd"
# Defines whether the load balancer for etcd will be internet facing or internal.
load_balancer_internal = false
# List of the security group IDs to apply to the load balancer (ingress TCP 2379) (if empty, defaults to open to all).
load_balancer_security_group_ids = []
# List of the security group IDs authorized to reach etcd/node-exporter metrics using the internal instances' IPs (if empty, metrics are not exposed)
metrics_security_group_ids = []

# Container image of ECO to use.
eco_image = "qmachu/etcd-cloud-operator:v3.3.3"
# Defines whether etcd should expect TLS clients connections.
eco_enable_tls = true
# Defines whether etcd should expect client certificates for client connections.
eco_require_client_certs = false
# Defines the interval between consecutive etcd snapshots (e.g. 30m).
eco_snapshot_interval = "30m"
# Defines the lifespan of each etcd snapshot (e.g. 24h).
eco_snapshot_ttl = "24h"
# Defines the maximum amount of data that etcd can store, in bytes, before going into maintenance mode
eco_backend_quota = "2147483648"
```

Finally, let Terraform configure and create the infrastructure:

```
cd terraform/platforms/aws
terraform init .
terraform apply .
```

After a few minutes, the etcd cluster will be available behind the endpoint
displayed. If client certificates authentication was enabled, they will be
displayed as well.

Here is a way to query and verify the health of the cluster:

```
export ETCDCTL_API=3

# Export the CA, if 'eco_enable_tls' was enabled.
terraform output ca > ca.crt; export ETCDCTL_CACERT=$(pwd)/ca.crt
export ETCDCTL_INSECURE_SKIP_TLS_VERIFY=true

# Export the client certs, if 'eco_require_client_certs' was enabled.
terraform output clients_cert > eco.crt; export ETCDCTL_CERT=$(pwd)/eco.crt
terraform output clients_key > eco.key; export ETCDCTL_KEY=$(pwd)/eco.key

etcdctl --endpoints=$(terraform output etcd_address) member list -w table
etcdctl --endpoints=$(terraform output etcd_address) endpoint status -w table
```

## Integrate the operator into your own project

Just like any other Terraform module, it can be used integrated into remote
projects. Simply add the following block in your code, with the configuration
relevant to your infrastructure:

```
module "eco" {
  source = "github.com/quentin-m/etcd-cloud-operator//terraform/platforms/aws?ref=v3.3.3"

  name = "eco-example"
  size = "3"

  instance_type         = "m5.large"
  instance_disk_size    = "30"
  instance_ssh_keys     = ["ssh-rsa ..."]

  associate_public_ip_address      = true
  subnets_ids                      = ["subnet-f438f793", "subnet-d4bea38c"]
  vpc_id                           = "vpc-19f019"
  route53_enabled                  = false
  route53_zone_id                  = ""
  load_balancer_internal           = false
  load_balancer_security_group_ids = []
  metrics_security_group_ids       = []

  eco_image                  = "qmachu/etcd-cloud-operator:v3.3.3"
  eco_enable_tls             = true
  eco_require_client_certs   = false
  eco_snapshot_interval      = "30m"
  eco_snapshot_ttl           = "24h"

  eco_backend_quota = "${2 * 1024 * 1024 * 1024}"

  ignition_extra_config = {
    source = "${local.ignition_extra_config}"
  }
}

// If you want to add extra ignition config, use like this
data "ignition_config" "extra" {
  users = [
    "${data.ignition_user.batman.id}",
  ]

  groups = [
    "${data.ignition_group.superheroes.id}",
  ]
}

data "ignition_group" "superheroes" {
  name = "superheroes"
}

data "ignition_user" "batman" {
    name = "batman"
    home_dir = "/home/batman/"
    shell = "/bin/bash"
}

// Alternatively, instead of using data-uri, you can host this on a web URl and pass that instead.
// See https://www.terraform.io/docs/providers/ignition/d/config.html#append
// for more details
locals {
  ignition_extra_config = "data:text/plain;charset=utf-8;base64,${base64encode(data.ignition_config.extra.rendered)}"
}
```

The following outputs will be available for use:

- `etcd_address`
- `ca`, `clients_cert`, `clients_key` (client certificate authentication)
- `instance_security_group` (Security Group ID for the instance SG)
