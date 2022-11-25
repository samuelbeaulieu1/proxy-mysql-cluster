data "aws_region" "current" {}

# get access to aws acc ID, user ID arn
data "aws_caller_identity" "this" {}

# Configure the Providers
provider "aws" {
  region = "us-east-1"
}

locals {
  app_id = "proxy-mysql-cluster"

  standalone_install_script_path    = "../standalone_install.sh"

  ubuntu22_ami  = "ami-08c40ec9ead489470"
  ubuntu20_ami  = "ami-0149b2da6ceec4bb0"

  sql_user = "app"
}

module "ec2_mysql_standalone" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "~> 3.0"

  name = "mysql-standalone"

  ami                         = local.ubuntu22_ami
  instance_type               = "t2.micro"
  key_name                    = module.key_pair.key_pair_name
  vpc_security_group_ids      = [module.sg.id]
  subnet_id                   = module.vpc.public_subnets[0]
  associate_public_ip_address = true

  user_data = templatefile("${local.standalone_install_script_path}", {
    REGION                   = var.region,
    SQL_PASSWORD             = var.sql_password
  })

  tags = {
    Terraform       = "true"
    Environment     = "dev"
  }
}