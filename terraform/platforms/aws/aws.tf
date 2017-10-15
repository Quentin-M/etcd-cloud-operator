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

data "aws_subnet" "main" {
  count = "${length(var.subnets_ids)}"
  id    = "${element(var.subnets_ids, count.index)}"
}

resource "aws_s3_bucket" "backups" {
  bucket = "${var.name}-backups"
  acl    = "private"
}

resource "aws_elb" "clients" {
  name = "${var.name}"

  internal                  = "${var.load_balancer_internal}"
  subnets                   = ["${data.aws_subnet.main.*.id}"]
  security_groups           = ["${aws_security_group.elb.id}"]
  cross_zone_load_balancing = true
  idle_timeout              = 3600

  listener {
    instance_port     = 2379
    instance_protocol = "tcp"
    lb_port           = 2379
    lb_protocol       = "tcp"
  }

  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 4
    timeout             = 5
    target              = "TCP:2379"
    interval            = 15
  }

  tags {
    Name = "${var.name}"
  }
}

resource "aws_security_group" "elb" {
  name   = "${var.name}-elb"
  vpc_id = "${data.aws_subnet.main.0.vpc_id}"

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

  tags = {
    "Name" = "${var.name}"
  }
}

locals {
  asg_provider        = "aws"
  snapshot_provider   = "s3"
  unseen_instance_ttl = "60s"
  snapshot_bucket     = "${aws_s3_bucket.backups.bucket}"
  advertise_address   = "${aws_elb.clients.dns_name}"
}
