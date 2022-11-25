#!/bin/bash

sudo apt-get update

mkdir -p /opt/mysqlcluster/home
cd /opt/mysqlcluster/home

wget http://dev.mysql.com/get/Downloads/MySQL-Cluster-7.2/mysql-cluster-gpl-7.2.1-linux2.6-x86_64.tar.gz
tar xvf mysql-cluster-gpl-7.2.1-linux2.6-x86_64.tar.gz
ln -s mysql-cluster-gpl-7.2.1-linux2.6-x86_64 mysqlc

tee -a /etc/profile.d/mysqlc.sh <<EOT
export MYSQLC_HOME=/opt/mysqlcluster/home/mysqlc
export PATH=\$MYSQLC_HOME/bin:\$PATH
EOT

source /etc/profile.d/mysqlc.sh
sudo apt-get update && sudo apt-get -y install libncurses5

mkdir -p /opt/mysqlcluster/deploy/ndb_data
ndbd -c ${MGMT_NODE_IP}:1186

cd /opt/mysqlcluster/deploy
mkdir conf
mkdir mysqld_data
cd conf

tee -a my.cnf <<EOT
[mysqld]
ndbcluster
datadir=/opt/mysqlcluster/deploy/mysqld_data
basedir=/opt/mysqlcluster/home/mysqlc
port=3306

[mysql_cluster]
ndb-connectstring=${MGMT_NODE_IP}
EOT

cd /opt/mysqlcluster/home/mysqlc
sudo scripts/mysql_install_db --no-defaults --datadir=/opt/mysqlcluster/deploy/mysqld_data

/opt/mysqlcluster/home/mysqlc/bin/mysqld --defaults-file=/opt/mysqlcluster/deploy/conf/my.cnf --user=root &

sleep 60

tee -a /home/ubuntu/setup_user.sql <<EOT
CREATE USER ${SQL_USER}@'%' IDENTIFIED BY '${SQL_PASSWORD}';
GRANT ALL PRIVILEGES ON *.* TO ${SQL_USER}@'%';
USE mysql;
DELETE FROM user WHERE user="" AND host="localhost";
FLUSH PRIVILEGES;
EOT

sudo /opt/mysqlcluster/home/mysqlc/bin/mysqladmin -u root password '${SQL_PASSWORD}'
mysql -u root -p${SQL_PASSWORD} < /home/ubuntu/setup_user.sql

rm -f /home/ubuntu/setup_user.sql