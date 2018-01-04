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

data "aws_ami" "coreos" {
  most_recent = true

  filter {
    name   = "name"
    values = ["CoreOS-stable-*"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "owner-id"
    values = ["595879546273"]
  }
}

resource "aws_autoscaling_group" "main" {
  name               = "${var.name}"

  max_size         = "${var.size}"
  min_size         = "${var.size}"
  desired_capacity = "${var.size}"

  load_balancers            = ["${aws_elb.clients.name}"]
  health_check_grace_period = 120
  health_check_type         = "ELB"

  vpc_zone_identifier  = ["${var.subnets_ids}"]
  launch_configuration = "${aws_launch_configuration.main.name}"

  tags = [
    {
      key                 = "Name"
      value               = "${var.name}"
      propagate_at_launch = true
    },
  ]
}

resource "aws_launch_configuration" "main" {
  name_prefix = "${var.name}"

  image_id      = "${data.aws_ami.coreos.image_id}"
  instance_type = "${var.instance_type}"
  key_name      = "${var.instance_ssh_key_name}"

  security_groups      = ["${aws_security_group.instances.id}"]
  iam_instance_profile = "${aws_iam_instance_profile.instances.arn}"
  user_data            = "${module.ignition.ignition}"

  associate_public_ip_address = "${var.associate_public_ips}"

  root_block_device {
    volume_size = "${var.instance_disk_size}"
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes        = ["image_id"]
  }
}

resource "aws_security_group" "instances" {
  name   = "instances.${var.name}"
  vpc_id = "${var.vpc_id}"

  ingress {
    from_port       = 2379
    to_port         = 2379
    protocol        = "tcp"
    security_groups = ["${aws_elb.clients.source_security_group_id}"]
    self            = true
  }

  ingress {
    from_port = 2380
    to_port   = 2380
    protocol  = "tcp"
    self      = true
  }

  ingress {
    from_port       = 9100
    to_port         = 9100
    protocol        = "tcp"
    security_groups = ["${var.metrics_security_group_ids}"]
    self            = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_iam_instance_profile" "instances" {
  name = "${var.name}"
  role = "${aws_iam_role.instances.name}"
}

resource "aws_iam_role" "instances" {
  name = "${var.name}"
  path = "/"

  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {
                "Service": "ec2.amazonaws.com"
            },
            "Effect": "Allow",
            "Sid": ""
        }
    ]
}
EOF
}

data "template_file" "policy" {
  template = "${file("${path.module}/policy.json")}"

  vars {
    bucket = "${aws_s3_bucket.backups.bucket}"
  }
}

resource "aws_iam_role_policy" "instances" {
  name   = "${var.name}"
  role   = "${aws_iam_role.instances.id}"
  policy = "${data.template_file.policy.rendered}"
}
