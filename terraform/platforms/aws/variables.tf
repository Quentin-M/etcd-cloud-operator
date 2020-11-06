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

variable "name" {
  description = "Name of the deployment"
}

variable "size" {
  description = "Number of etcd members (must be odd)"
}

variable "instance_ami_id" {
  description = "ID of the AMI to use for the EC2 instances (if empty, defaults to Flatcar Linux)"
  default     = ""
}

variable "instance_type" {
  description = "Type of the EC2 instances to launch"
}

variable "instance_disk_size" {
  description = "Size of the disk associated to the EC2 instances (in GB)"
}

variable "associate_public_ips" {
  description = "Defines whether public IPs should be assigned to the EC2 instances (mainly depends if public or private subnets are used)"
}

variable "subnets_ids" {
  description = "List of the subnet IDs to place the EC2 instances in (should span across AZs for availability)"
  type        = list(string)
}

variable "vpc_id" {
  description = "ID of the VPC where the subnets are defined."
}

variable "route53_prefix" {
  description = "Defines the primary dns prefix for the Route53 entry"
  default = "etcd"
}

variable "route53_enabled" {
  description = "Defines whether a Route53 record should be created for client connections"
  default     = "false"
}

variable "route53_zone_id" {
  description = "Optional Route53 Zone ID under which an 'etcd' record should be created for client connections"
}

variable "load_balancer_internal" {
  description = "Defines whether the load balancer for etcd should be internet facing or internal"
}

variable "load_balancer_security_group_ids" {
  description = "List of the security group IDs to apply to the load balancer (ingress TCP 2379) (if empty, defaults to open to all)"
  type        = list(string)
  default     = []
}

variable "metrics_security_group_ids" {
  description = "List of the security group IDs authorized to reach etcd/node-exporter metrics using the internal instances' IPs (if empty, metrics are not exposed)"
  type        = list(string)
  default     = []
}

variable "ignition_extra_config" {
  description = "Extra ignition configuration that will get appended to the default ECO config"
  default     = {}
  type        = map(string)
}

variable "extra_tags" {
  type        = map(string)
  description = "Extra tags to associated with any AWS resources."
  default     = {}
}

variable "telegraf_graphite_uri" {
  description = "Defines graphite host to relay Telegraf metrics to"
}

variable "telegraf_graphite_prefix" {
  description = "Defines graphite prefix for relaying metrics"
  default = "eco"
}
