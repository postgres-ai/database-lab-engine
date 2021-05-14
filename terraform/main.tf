terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = "${var.aws_region}"
}
locals {
  common_tags = {
    Name = "${var.tag_name}"
  }
}
