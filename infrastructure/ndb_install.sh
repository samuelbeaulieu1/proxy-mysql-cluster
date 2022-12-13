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