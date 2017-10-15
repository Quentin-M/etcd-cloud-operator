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

variable "common_name" {
  description = "Defines the common name for the clients_server certificate."
}

variable "generate_clients_cert" {
  description = "Defines whether additional clients certificates should be generated in addition to server certificates"
}
