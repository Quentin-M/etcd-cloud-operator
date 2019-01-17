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

# CA.

resource "tls_private_key" "ca" {
  count = "${var.enabled == true && length(var.ca["key"]) == 0 ? 1 : 0}"

  algorithm = "RSA"
  rsa_bits  = "2048"
}

resource "tls_self_signed_cert" "ca" {
  count = "${var.enabled == true && length(var.ca["key"]) == 0 ? 1 : 0}"

  key_algorithm         = "${tls_private_key.ca.algorithm}"
  private_key_pem       = "${tls_private_key.ca.private_key_pem}"
  is_ca_certificate     = true
  validity_period_hours = 8760

  subject {
    common_name  = "etcd"
    organization = "etcd"
  }

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "cert_signing",
  ]
}

locals {
  ca = {
    "cert" = "${length(var.ca["key"]) == 0 ? join("", tls_self_signed_cert.ca.*.cert_pem) : var.ca["cert"]}"
    "key"  = "${length(var.ca["key"]) == 0 ? join("", tls_private_key.ca.*.private_key_pem) : var.ca["key"]}"
    "alg"  = "${length(var.ca["key"]) == 0 ? join("", tls_private_key.ca.*.algorithm) : var.ca["alg"]}"
  }
}

# Certificate for the etcd client server.

resource "tls_private_key" "clients-server" {
  count = "${var.enabled == true ? 1 : 0}"

  algorithm = "RSA"
  rsa_bits  = "2048"
}

resource "tls_cert_request" "clients-server" {
  count = "${var.enabled == true ? 1 : 0}"

  key_algorithm   = "${tls_private_key.clients-server.algorithm}"
  private_key_pem = "${tls_private_key.clients-server.private_key_pem}"

  subject {
    common_name  = "${var.common_name}"
    organization = "etcd"
  }
}

resource "tls_locally_signed_cert" "clients-server" {
  count = "${var.enabled == true ? 1 : 0}"

  cert_request_pem      = "${tls_cert_request.clients-server.cert_request_pem}"
  ca_key_algorithm      = "${local.ca["alg"]}"
  ca_private_key_pem    = "${local.ca["key"]}"
  ca_cert_pem           = "${local.ca["cert"]}"
  validity_period_hours = 8760

  allowed_uses = [
    "key_encipherment",
    "client_auth",
    "server_auth",
  ]
}

# Certificates for the etcd clients.

resource "tls_private_key" "clients" {
  count = "${var.enabled == true && var.generate_clients_cert == true ? 1 : 0}"

  algorithm = "RSA"
  rsa_bits  = "2048"
}

resource "tls_cert_request" "clients" {
  count = "${var.enabled == true && var.generate_clients_cert == true ? 1 : 0}"

  key_algorithm   = "${tls_private_key.clients.algorithm}"
  private_key_pem = "${tls_private_key.clients.private_key_pem}"

  subject {
    common_name  = "etcd"
    organization = "etcd"
  }
}

resource "tls_locally_signed_cert" "clients" {
  count = "${var.enabled == true && var.generate_clients_cert == true ? 1 : 0}"

  cert_request_pem      = "${tls_cert_request.clients.cert_request_pem}"
  ca_key_algorithm      = "${local.ca["alg"]}"
  ca_private_key_pem    = "${local.ca["key"]}"
  ca_cert_pem           = "${local.ca["cert"]}"
  validity_period_hours = 8760

  allowed_uses = [
    "key_encipherment",
    "client_auth",
  ]
}
