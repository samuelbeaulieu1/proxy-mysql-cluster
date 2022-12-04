module "common_sg" {
  source     = "cloudposse/security-group/aws"
  attributes = ["common"]

  # Allow unlimited egress
  allow_all_egress = true

  rules = [
    # Allow ssh
    {
      key         = "ssh"
      type        = "ingress"
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      self        = null
      description = "Allow SSH from anywhere"
    }
  ]

  vpc_id = module.vpc.vpc_id
}

module "cluster_sg" {
  source     = "cloudposse/security-group/aws"
  attributes = ["cluster"]

  # Allow unlimited egress
  allow_all_egress = true

  rules = [
    # Allow all tcp communication from within the MySQL internal network
    # NDB nodes open multiple ports to communicate
    {
      key         = "ALL TCP FROM INTERNAL"
      type        = "ingress"
      from_port   = 0
      to_port     = 65535
      protocol    = "tcp"
      cidr_blocks = ["10.0.10.0/24"]
      self        = null
      description = "Allow all TCP from internal"
    },
    # Allow to ping instances from within the cluster's internal network
    {
      key         = "ICMP"
      type        = "ingress"
      from_port   = -1
      to_port     = -1
      protocol    = "ICMP"
      cidr_blocks = ["10.0.10.0/24"]
      self        = null
      description = "Allow ping"
    }
  ]

  vpc_id = module.vpc.vpc_id
}

module "mysql_sg" {
  source     = "cloudposse/security-group/aws"
  attributes = ["mysql"]

  # Allow unlimited egress
  allow_all_egress = true

  rules = [
    # Allow MySQL from outside the network 
    {
      key         = "MYSQL"
      type        = "ingress"
      from_port   = 3306
      to_port     = 3306
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      self        = null
      description = "Allow MYSQL"
    }
  ]

  vpc_id = module.vpc.vpc_id
}