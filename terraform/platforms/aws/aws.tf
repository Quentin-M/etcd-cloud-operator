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

resource "aws_s3_bucket" "backups" {
  bucket = "${var.name}"
  acl    = "private"

  tags = "${merge(map(
    "Name", "${var.name}",
    "owner", "etcd-cloud-operator"
  ), var.extra_tags)}"
}

resource "aws_elb" "clients" {
  name = "${replace(var.name, "/[^a-zA-Z0-9-]/", "-")}"

  internal = "${var.load_balancer_internal}"
  subnets  = ["${var.subnets_ids}"]

  security_groups = [
    "${split(",", length(var.load_balancer_security_group_ids) > 0
    ? join(",", var.load_balancer_security_group_ids)
    : aws_security_group.elb.id)}",
  ]

  cross_zone_load_balancing = true
  idle_timeout              = 3600

  listener {
    instance_port     = 2379
    instance_protocol = "tcp"
    lb_port           = 2379
    lb_protocol       = "tcp"
  }

  health_check {
    healthy_threshold   = 6
    unhealthy_threshold = 2
    timeout             = 3
    target              = "TCP:2379"
    interval            = 5
  }
}

resource "aws_security_group" "elb" {
  # ERR: aws_security_group.elb: value of 'count' cannot be computed
  # REF: https://github.com/hashicorp/terraform/issues/4149
  #
  # count  = "${length(var.load_balancer_security_group_ids) > 0 ? 0 : 1}"
  name = "default.elb.${var.name}"

  vpc_id = "${var.vpc_id}"

  ingress {
    from_port   = 2379
    to_port     = 2379
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 2379
    to_port     = 2379
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

locals {
  asg_provider         = "aws"
  snapshot_provider    = "s3"
  unhealthy_member_ttl = "3m"
  snapshot_bucket      = "${aws_s3_bucket.backups.bucket}"
  advertise_address    = "${var.route53_enabled == "true" ? join("", aws_route53_record.elb.*.name) : aws_elb.clients.dns_name}"
}
