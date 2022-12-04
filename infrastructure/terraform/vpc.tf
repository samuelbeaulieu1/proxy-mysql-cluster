# Custom VPC for all EC2 instances, both cluster and stand-alone
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  name = "vpc-mysql-cluster"

  cidr = "10.0.0.0/16"

  azs             = ["${data.aws_region.current.name}a", "${data.aws_region.current.name}b"]
  private_subnets = ["10.0.0.0/24", "10.0.1.0/24"]
  public_subnets  = ["10.0.10.0/24", "10.0.11.0/24"]

  enable_nat_gateway         = true
  enable_dns_hostnames       = true
  enable_dns_support         = true
  manage_default_route_table = true

  tags = {
    Terraform   = "true"
    Environment = "dev"
  }
}