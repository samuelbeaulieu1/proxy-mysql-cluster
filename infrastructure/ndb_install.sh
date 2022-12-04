#!/bin/bash

sudo apt-get update

mkdir -p /opt/mysqlcluster/home
cd /opt/mysqlcluster/home

# Download MySQL cluster package
wget http://dev.mysql.com/get/Downloads/MySQL-Cluster-7.2/mysql-cluster-gpl-7.2.1-linux2.6-x86_64.tar.gz
tar xvf mysql-cluster-gpl-7.2.1-linux2.6-x86_64.tar.gz
ln -s mysql-cluster-gpl-7.2.1-linux2.6-x86_64 mysqlc

# Env vars for mysql binaries
tee -a /etc/profile.d/mysqlc.sh <<EOT
export MYSQLC_HOME=/opt/mysqlcluster/home/mysqlc
export PATH=\$MYSQLC_HOME/bin:\$PATH
EOT

# Setting env vars for current install script
source /etc/profile.d/mysqlc.sh
sudo apt-get update && sudo apt-get -y install libncurses5

# Starting NDB node and connect to management node with private IP
mkdir -p /opt/mysqlcluster/deploy/ndb_data
ndbd -c ${MGMT_NODE_IP}:1186

# Creating data/config folders for mysql cluster
cd /opt/mysqlcluster/deploy
mkdir conf
mkdir mysqld_data
cd conf

# MySQL Config for data node
tee -a my.cnf <<EOT
[mysqld]
ndbcluster
datadir=/opt/mysqlcluster/deploy/mysqld_data
basedir=/opt/mysqlcluster/home/mysqlc
port=3306

[mysql_cluster]
ndb-connectstring=${MGMT_NODE_IP}
EOT

# Install base MySQL
cd /opt/mysqlcluster/home/mysqlc
sudo scripts/mysql_install_db --no-defaults --datadir=/opt/mysqlcluster/deploy/mysqld_data

# Start MySQL
/opt/mysqlcluster/home/mysqlc/bin/mysqld --defaults-file=/opt/mysqlcluster/deploy/conf/my.cnf --user=root &

sleep 60

# Setup remote user
tee -a /home/ubuntu/setup_user.sql <<EOT
CREATE USER ${SQL_USER}@'%' IDENTIFIED BY '${SQL_PASSWORD}';
GRANT ALL PRIVILEGES ON *.* TO ${SQL_USER}@'%';
USE mysql;
DELETE FROM user WHERE user="" AND host="localhost";
FLUSH PRIVILEGES;
EOT

# Change password and install remote user
sudo /opt/mysqlcluster/home/mysqlc/bin/mysqladmin -u root password '${SQL_PASSWORD}'
mysql -u root -p${SQL_PASSWORD} < /home/ubuntu/setup_user.sql

rm -f /home/ubuntu/setup_user.sql