
terraform {
  required_version = ">= 0.13"
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    ignition = {
      source = "terraform-providers/ignition"
    }
    template = {
      source = "hashicorp/template"
    }
  }
}
