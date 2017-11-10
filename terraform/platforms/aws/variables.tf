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

variable "max_size" {
  description = "Number of max etcd members in ASG (must be odd)"
}

variable "min_size" {
  description = "Number of min etcd members in ASG (must be odd)"
}

variable "desired_capacity" {
  description = "Number of desired etcd members in ASG (must be odd)"
}

variable "min_elb_capacity" {
  description = "Number of minimum elb capacity for ASG (must be same as min_size)"
}

variable "instance_type" {
  description = "Type of the EC2 instances to launch"
}

variable "instance_disk_size" {
  description = "Size of the disk associated to the EC2 instances (in GB)"
}

variable "instance_ssh_key_name" {
  description = "Name of the SSH key to use (must be present on EC2)"
}

variable "associate_public_ips" {
  description = "Defines whether public IPs should be assigned to the EC2 instances (mainly depends if public or private subnets are used)"
}

variable "subnets_ids" {
  description = "List of the subnet IDs to place the EC2 instances in (should span across AZs for availability)"
  type        = "list"
}

variable "vpc_id" {
  description = "ID of the VPC where the subnets are defined."
}

variable "load_balancer_internal" {
  description = "Defines whether the load balancer for etcd should be internet facing or internal"
}

variable "load_balancer_security_group_ids" {
  description = "List of the security group IDs to apply to the load balancer (ingress TCP 2379) (if empty, defaults to open to all)"
  type        = "list"
  default     = []
}
