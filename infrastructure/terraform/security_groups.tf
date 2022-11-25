module "sg" {
  source     = "cloudposse/security-group/aws"
  attributes = ["primary"]

  # Allow unlimited egress
  allow_all_egress = true

  rules = [
    {
      key         = "ssh"
      type        = "ingress"
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      self        = null
      description = "Allow SSH from anywhere"
    },
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
    {
      key         = "MYSQL"
      type        = "ingress"
      from_port   = 3306
      to_port     = 3306
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      self        = null
      description = "Allow MYSQL"
    },
    {
      key         = "ICMP"
      type        = "ingress"
      from_port   = -1
      to_port     = -1
      protocol    = "ICMP"
      cidr_blocks = ["0.0.0.0/0"]
      self        = null
      description = "Allow ping"
    }
  ]

  vpc_id = module.vpc.vpc_id
}