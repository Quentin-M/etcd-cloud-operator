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

variable "eco_unhealthy_member_ttl" {
  description = "Defines how long an unhealthy member will be kept as a member in the cluster"
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

variable "eco_backend_quota" {
  description = "Defines the maximum amount of data that etcd can store, in bytes, before going into maintenance mode"
}

variable "eco_auto_compaction_mode" {
  description = "Defines the auto-compaction mode (periodic, or revision)."
}

variable "eco_auto_compaction_retention" {
  description = "Defines the auto-compaction retention (e.g. 5min for periodic based compaction, or a revision number. set to 0 to disable)."
}

variable "eco_config_file" {
  description = "Defines the content of the eco config file, if not empty, then will use this config file directly instead of of the config file templates (optional)"
}

variable "telegraf_config_file" {
  description = "Defines the content of the Telegraf config file"
}

variable "telegraf_outputs_config" {
  description = "Defines output(s) to append to default Telegraf config"
}

variable "eco_init_acl_rootpw" {
  description = "Defines the root passsword of the etcd (optional)"
}

variable "eco_init_acl_roles" {
  description = "Defines the list of ACL roles for the etcd (optional)"
}

variable "eco_init_acl_users" {
  description = "Defines the list of ACL users for the etcd (optional)"
}
