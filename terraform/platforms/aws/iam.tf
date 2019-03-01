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
    bucket     = "${aws_s3_bucket.backups.bucket}"
    kms_arn    = "${aws_kms_key.key.arn}"
    config_arn = "arn:aws:s3:::${aws_s3_bucket.config.id}/${aws_s3_bucket_object.ignition_config.id}"
  }
}

resource "aws_iam_role_policy" "instances" {
  name   = "${var.name}"
  role   = "${aws_iam_role.instances.id}"
  policy = "${data.template_file.policy.rendered}"
}
