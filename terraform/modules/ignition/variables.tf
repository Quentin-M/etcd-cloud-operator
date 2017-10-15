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

variable "eco_image" {
  description = "Defines the container image to run for ECO"
}

variable "eco_cert" {
  description = "Defines the certificate to use to encrypt client connections on etcd (optional)"
}

variable "eco_key" {
  description = "Defines the private key to use to encrypt client connections on etcd (optional)"
}

variable "eco_ca" {
  description = "Defines the CA to use to encrypt client connections on etcd (optional)"
}

variable "eco_configuration" {
  description = "Defines the configuration for ECO"
}
