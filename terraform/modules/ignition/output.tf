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

output "ignition" {
  value = "${data.ignition_config.main.rendered}"
}

output "eco_ca_file" {
  value = "/etc/eco/ca.crt"
}

output "eco_cert_file" {
  value = "/etc/eco/eco.crt"
}

output "eco_key_file" {
  value = "/etc/eco/eco.key"
}
