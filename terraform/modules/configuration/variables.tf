// Copyright 2017 Quentin Machu & eco authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

variable "eco_asg_provider" {
  description = "Defines the auto-scaling provider for ECO"
}

variable "eco_snapshot_provider" {
  description = "Defines the snapshot provider for ECO"
}

variable "eco_unseen_instance_ttl" {
  description = "Defines how long a member, which instance has not been seen, is tolerated in the cluster"
}

variable "eco_enable_auto_disaster_recovery" {
  description = "Defines whether automatic disaster recovery on ECO should be enabled"
}

variable "eco_advertise_address" {
  description = "Defines the address advertised to reach etcd (unset means the instances' private IPs will be used)"
}

variable "eco_cert_file" {
  description = "Defines the path to the certificate to use to encrypt client connections on etcd (optional)"
}

variable "eco_key_file" {
  description = "Defines the path to the private key to use to encrypt client connections on etcd (optional)"
}

variable "eco_ca_file" {
  description = "Defines the path to the CA to use to encrypt client connections on etcd (optional)"
}

variable "eco_require_client_cert" {
  description = "Defines whether client certificates are required to establish client connections on etcd"
}

variable "eco_snapshot_interval" {
  description = "Defines the interval between consecutive etcd snapshots"
}

variable "eco_snapshot_ttl" {
  description = "Defines the lifespan of each etcd snapshot"
}

variable "eco_snapshot_bucket" {
  description = "Defines the S3 Bucket location for etcd snapshot when the AWS provider is used on ECO"
}
