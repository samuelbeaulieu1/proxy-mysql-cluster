output "mysql-standalone-domain" {
    description = "MYSQL standalone setup domain name"
    value       = module.ec2_mysql_standalone.public_dns
}

output "mysql-mgmt-domain" {
    description = "MYSQL ndb management setup domain name"
    value       = module.ec2_mysql_cluster_mgmt.public_dns
}

output "mysql-ndb-domain" {
    description = "MYSQL ndb domain name"
    value       = [
        for node in local.data_nodes : module.ec2_mysql_cluster_ndb[node].public_dns
    ]
}