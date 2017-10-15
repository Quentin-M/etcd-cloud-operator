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

// This file is designed to be included in each supported platforms'
// directories. It will then automatically expose ECO variables, generated
// configuration file, ignition config and common output (such as the CA, and
// clients certs if enabled). It requires the implementing platform to define
// the following locals, as they are platform specific.
//
// locals {
//   asg_provider = ""
//   snapshot_provider = ""
//   unseen_instance_ttl
//   advertise_address = ""
//   snapshot_bucket = ""
// }

// Variables.

variable "eco_image" {
  description = "Container image of ECO to use"
  default     = "qmachu/etcd-cloud-operator-dev:latest"
}

variable "eco_enable_tls" {
  description = "Defines whether etcd should expect TLS clients connections"
  default     = true
}

variable "eco_require_client_certs" {
  description = "Defines whether etcd should expect client certificates for client connections"
}

variable "eco_auto_disaster_recovery" {
  description = "Defines whether automatic disaster recovery on ECO should be enabled"
}

variable "eco_snapshot_interval" {
  description = "Defines the interval between consecutive etcd snapshots (e.g. 30m)"
}

variable "eco_snapshot_ttl" {
  description = "Defines the lifespan of each etcd snapshot (e.g. 24h)"
}

// Modules.

module "tls" {
  source = "../../modules/tls"

  enabled               = "${var.eco_enable_tls}"
  common_name           = "${local.advertise_address}"
  generate_clients_cert = "${var.eco_require_client_certs}"
}

module "configuration" {
  source = "../../modules/configuration"

  eco_asg_provider      = "${local.asg_provider}"
  eco_snapshot_provider = "${local.snapshot_provider}"

  eco_unseen_instance_ttl = "${local.unseen_instance_ttl}"
  eco_advertise_address   = "${local.advertise_address}"
  eco_snapshot_bucket     = "${local.snapshot_bucket}"

  eco_ca_file             = "${var.eco_enable_tls == true ? module.ignition.eco_ca_file : ""}"
  eco_cert_file           = "${var.eco_enable_tls == true ? module.ignition.eco_cert_file : ""}"
  eco_key_file            = "${var.eco_enable_tls == true ? module.ignition.eco_key_file : ""}"
  eco_require_client_cert = "${var.eco_require_client_certs}"

  eco_enable_auto_disaster_recovery = "${var.eco_auto_disaster_recovery}"
  eco_snapshot_interval             = "${var.eco_snapshot_interval}"
  eco_snapshot_ttl                  = "${var.eco_snapshot_ttl}"
}

module "ignition" {
  source = "../../modules/ignition"

  eco_image         = "${var.eco_image}"
  eco_configuration = "${module.configuration.configuration}"

  eco_cert = "${module.tls.clients_server_cert}"
  eco_key  = "${module.tls.clients_server_key}"
  eco_ca   = "${module.tls.ca}"
}

// Output.

output "ca" {
  value = "${module.tls.ca}"
}

output "clients_cert" {
  value = "${module.tls.clients_cert}"
}

output "clients_key" {
  value = "${module.tls.clients_key}"
}
