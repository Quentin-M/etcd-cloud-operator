// This bucket is used for saving the ignition config
resource "aws_s3_bucket" "config" {
  bucket = "${var.name}-config"
  acl    = "private"

  versioning {
    enabled = true
  }

  tags = merge(
    {
      "Name"       = "${var.name}-config"
      "managed_by" = "etcd-cloud-operator"
    },
    var.extra_tags,
  )

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.key.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
}

resource "aws_kms_key" "key" {
  description         = var.name
  enable_key_rotation = true

  tags = merge(
    {
      "Name"       = var.name
      "managed_by" = "etcd-cloud-operator"
    },
    var.extra_tags,
  )
}

resource "aws_s3_bucket_object" "ignition_config" {
  key        = "config.json"
  bucket     = aws_s3_bucket.config.id
  content    = module.ignition.ignition
  kms_key_id = aws_kms_key.key.arn
}

resource "aws_s3_bucket" "backups" {
  bucket = var.name
  acl    = "private"

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.key.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }

  tags = merge(
    {
      "Name"       = var.name
      "managed_by" = "etcd-cloud-operator"
    },
    var.extra_tags,
  )
}

