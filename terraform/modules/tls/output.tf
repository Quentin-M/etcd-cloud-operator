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

output "ca" {
  value = "${var.enabled == true ? tls_self_signed_cert.ca.cert_pem : ""}"
}

output "clients_server_cert" {
  value = "${var.enabled == true ? tls_locally_signed_cert.clients-server.cert_pem : ""}"
}

output "clients_server_key" {
  value = "${var.enabled == true ? tls_private_key.clients-server.private_key_pem : ""}"
}

output "clients_cert" {
  value = "${var.enabled == true && var.generate_clients_cert == true ? tls_locally_signed_cert.clients.cert_pem : ""}"
}

output "clients_key" {
  value = "${var.enabled == true && var.generate_clients_cert == true ? tls_private_key.clients.private_key_pem : ""}"
}
