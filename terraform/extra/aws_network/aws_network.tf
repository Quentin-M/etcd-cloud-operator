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

data "aws_availability_zones" "azs" {}

resource "aws_vpc" "main" {
  cidr_block           = "172.16.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    "Name" = "${var.name}"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = "${aws_vpc.main.id}"

  tags {
    "Name" = "${var.name}"
  }
}

resource "aws_subnet" "public" {
  count = "${length(data.aws_availability_zones.azs.names)}"

  availability_zone = "${data.aws_availability_zones.azs.names[count.index]}"
  cidr_block        = "${cidrsubnet(aws_vpc.main.cidr_block, 4, count.index)}"
  vpc_id            = "${aws_vpc.main.id}"

  tags = {
    "Name" = "public.${data.aws_availability_zones.azs.names[count.index]}.${var.name}"
  }
}

resource "aws_route_table_association" "public" {
  count = "${aws_subnet.public.count}"

  route_table_id = "${aws_route_table.public.*.id[count.index]}"
  subnet_id      = "${aws_subnet.public.*.id[count.index]}"
}

resource "aws_route_table" "public" {
  count  = "${aws_subnet.public.count}"
  vpc_id = "${aws_vpc.main.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.main.id}"
  }

  tags = {
    "Name" = "${data.aws_availability_zones.azs.names[count.index]}.${var.name}"
  }
}
