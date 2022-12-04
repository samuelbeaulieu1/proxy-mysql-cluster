# SSH keys to connect to EC2 instances
module "key_pair" {
  source = "terraform-aws-modules/key-pair/aws"

  key_name           = local.app_id
  create_private_key = true
}