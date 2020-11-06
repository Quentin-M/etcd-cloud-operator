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

locals {
  eco_configuration = <<EOT
eco:
  unhealthy-member-ttl: ${var.eco_unhealthy_member_ttl}
  etcd:
    advertise-address: ${var.eco_advertise_address}
    data-dir: /var/lib/eco/etcd
    client-transport-security:
      auto-tls: false
      cert-file: ${var.eco_cert_file}
      key-file: ${var.eco_key_file}
      trusted-ca-file: ${var.eco_ca_file}
      client-cert-auth: ${var.eco_require_client_cert}
    peer-transport-security:
      auto-tls: true
    backend-quota: ${var.eco_backend_quota}
    auto-compaction-mode: ${var.eco_auto_compaction_mode}
    auto-compaction-retention: ${var.eco_auto_compaction_retention}
%{ if length(var.eco_init_acl_users) > 0 ~}
    init-acl:
%{ if var.eco_init_acl_rootpw != "" ~}
      rootPassword: ${var.eco_init_acl_rootpw}
%{ endif ~}
      roles:
%{ for role in var.eco_init_acl_roles ~}
      - name: ${role.name}
        permissions:
%{ for perm in role.permissions ~}
        - mode: ${perm.mode}
          key: ${perm.key}
          prefix: ${perm.prefix}
%{ if perm.rangeEnd != "" ~}
          rangeEnd: ${perm.rangeEnd}
%{ endif ~}
%{ endfor ~}
%{ endfor ~}
      users:
%{ for user in var.eco_init_acl_users ~}
      - name: ${user.name}
%{ if user.password != "" ~}
        password: ${user.password}
%{ endif ~}
        roles:
%{ for role in user.roles ~}
        - ${role}
%{ endfor ~}
%{ endfor ~}
%{ endif ~}
  asg:
    provider: ${var.eco_asg_provider}
  snapshot:
    provider: ${var.eco_snapshot_provider}
    interval: ${var.eco_snapshot_interval}
    ttl: ${var.eco_snapshot_ttl}
    bucket: ${var.eco_snapshot_bucket}
EOT
  telegraf_configuration = <<EOT
[inputs.prometheus]
## An array of urls to scrape metrics from.
urls = ["http://localhost:2381/metrics"]
metric_version = 2

${var.telegraf_outputs_config}
EOT
}
