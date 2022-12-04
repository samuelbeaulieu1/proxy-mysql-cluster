variable "build_version" {
  type    = string
  default = "1.0.0"
}

# AWS region
variable "region" {
  type    = string
  default = "us-east-1"
}

# SQL password for cluster and stand-alone
variable "sql_password" {
  type    = string
  default = "123"
}
