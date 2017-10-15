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

data "template_file" "configuration" {
  template = "${file("${path.module}/config.yaml")}"

  vars {
    asg_provider      = "${var.eco_asg_provider}"
    snapshot_provider = "${var.eco_snapshot_provider}"

    unseen_instance_ttl           = "${var.eco_unseen_instance_ttl}"
    enable_auto_disaster_recovery = "${var.eco_enable_auto_disaster_recovery}"
    advertise_address             = "${var.eco_advertise_address}"

    cert_file           = "${var.eco_cert_file}"
    key_file            = "${var.eco_key_file}"
    ca_file             = "${var.eco_ca_file}"
    require_client_cert = "${var.eco_require_client_cert}"

    snapshot_interval = "${var.eco_snapshot_interval}"
    snapshot_ttl      = "${var.eco_snapshot_ttl}"
    snapshot_bucket   = "${var.eco_snapshot_bucket}"
  }
}
