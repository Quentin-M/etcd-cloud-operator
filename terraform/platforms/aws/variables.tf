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

variable "instance_type" {
  description = "Type of the EC2 instances to launch"
  default     = ""
}

variable "instance_disk_size" {
  description = "Size of the disk associated to the EC2 instances (in GB)"
}

variable "instance_ssh_key_name" {
  description = "Name of the SSH key to use (must be present on EC2)"
}

variable "subnets_ids" {
  description = "List of the subnet IDs to place the EC2 instances in (should span across AZs for availability)"
  type        = "list"
}

variable "load_balancer_internal" {
  description = "Defines whether the load balancer for etcd will be internet facing or internal"
}
