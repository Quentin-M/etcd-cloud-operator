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

variable "enabled" {
  description = "Defines whether the module is enabled or not"
}

variable "ca" {
  description = "Optional CA keypair from which all certificates should be generated ('cert', 'key', 'alg')"
  type        = map(string)
}

variable "common_name" {
  description = "Defines the common name for the clients_server certificate."
}

variable "san_dns_names" {
  description = "List of additional DNS names for which the server certificate has to be trusted"
  type        = list(string)
}

variable "generate_clients_cert" {
  description = "Defines whether additional clients certificates should be generated in addition to server certificates"
}

variable "eco_init_acl_users" {
  description = "Defines the list of ACL users for the etcd (optional)"
}

variable "enable_jwt_token" {
  description = "Defines whether or not to create private/public key pairs to verify jwt tokens"
  default     = false
}
