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
  cluster_mgmt_install_script_path  = "../mgmt_install.sh"
  cluster_ndb_install_script_path   = "../ndb_install.sh"

  ubuntu22_ami  = "ami-08c40ec9ead489470"
  ubuntu20_ami  = "ami-0149b2da6ceec4bb0"

  data_ips      = ["ip-10-0-10-26.ec2.internal", "ip-10-0-10-27.ec2.internal", "ip-10-0-10-28.ec2.internal"]
  data_nodes    = toset(["10.0.10.26", "10.0.10.27", "10.0.10.28"])
  mgmt_node_ip  = "10.0.10.25"
  mgmt_node_dns = "ip-10-0-10-25.ec2.internal"

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

module "ec2_mysql_cluster_mgmt" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "~> 3.0"

  name = "mysql-mgmt"

  ami                         = local.ubuntu20_ami
  instance_type               = "t2.micro"
  key_name                    = module.key_pair.key_pair_name
  vpc_security_group_ids      = [module.sg.id]
  subnet_id                   = module.vpc.public_subnets[0]
  private_ip                  = local.mgmt_node_ip
  associate_public_ip_address = true

  user_data = templatefile("${local.cluster_mgmt_install_script_path}", {
    SQL_PASSWORD             = var.sql_password,
    SQL_USER                 = local.sql_user,
    MGMT_NODE_IP             = local.mgmt_node_dns,
    DATA_NODE1_IP            = local.data_ips[0],
    DATA_NODE2_IP            = local.data_ips[1],
    DATA_NODE3_IP            = local.data_ips[2]
  })

  tags = {
    Terraform       = "true"
    Environment     = "dev"
  }
}

module "ec2_mysql_cluster_ndb" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "~> 3.0"

  for_each = local.data_nodes

  name = "mysql-ndb-${each.key}"

  ami                         = local.ubuntu20_ami
  instance_type               = "t2.micro"
  key_name                    = module.key_pair.key_pair_name
  vpc_security_group_ids      = [module.sg.id]
  subnet_id                   = module.vpc.public_subnets[0]
  private_ip                  = each.key
  associate_public_ip_address = true

  user_data = templatefile("${local.cluster_ndb_install_script_path}", {
    SQL_PASSWORD             = var.sql_password,
    SQL_USER                 = local.sql_user,
    MGMT_NODE_IP             = module.ec2_mysql_cluster_mgmt.private_dns
  })

  tags = {
    Terraform       = "true"
    Environment     = "dev"
  }
}