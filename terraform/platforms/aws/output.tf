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

output "etcd_address" {
  value = "${var.eco_enable_tls == true ? "https" : "http"}://${var.route53_zone_id != "" ? join("", aws_route53_record.elb.*.name) : aws_elb.clients.dns_name}:2379"
}

// You can attach extra rules using aws_security_group_rule
output "instance_security_group" {
  value = aws_security_group.instances.id
}

output "lb_dns_name" {
  value = aws_elb.clients.dns_name
}

output "lb_zone_id" {
  value = aws_elb.clients.zone_id
}

