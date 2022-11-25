data "aws_region" "current" {}

# get access to aws acc ID, user ID arn
data "aws_caller_identity" "this" {}

# Configure the Providers
provider "aws" {
  region = "us-east-1"
}

locals {
  app_id = "proxy-mysql-cluster"

  ubuntu22_ami  = "ami-08c40ec9ead489470"
  ubuntu20_ami  = "ami-0149b2da6ceec4bb0"
}