// This bucket is used for saving the ignition config
resource "aws_s3_bucket" "config" {
  bucket = "${var.name}-config"
  acl    = "private"

  versioning {
    enabled = true
  }

  tags = "${merge(map(
    "Name", "${var.name}-config",
    "owner", "etcd-cloud-operator"
  ), var.extra_tags)}"
}

resource "aws_kms_key" "key" {
  description         = "${var.name}"
  enable_key_rotation = true

  tags = "${merge(map(
    "Name", "${var.name}",
    "owner", "etcd-cloud-operator"
  ), var.extra_tags)}"
}

resource "aws_s3_bucket_object" "ignition_config" {
  key        = "config.json"
  bucket     = "${aws_s3_bucket.config.id}"
  content    = "${module.ignition.ignition}"
  kms_key_id = "${aws_kms_key.key.arn}"
}

data "ignition_config" "s3" {
  replace {
    source       = "${format("s3://%s/%s", "${aws_s3_bucket.config.id}", aws_s3_bucket_object.ignition_config.key)}"
    verification = "sha512-${sha512(module.ignition.ignition)}"
  }
}
