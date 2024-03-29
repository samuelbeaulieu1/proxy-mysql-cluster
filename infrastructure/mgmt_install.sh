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

# Creating data/config/deployment folders for mysql cluster
mkdir -p /opt/mysqlcluster/deploy
cd /opt/mysqlcluster/deploy
mkdir conf
mkdir mysqld_data
mkdir ndb_data
cd conf

# Master MySQL config
tee -a my.cnf <<EOT
[mysqld]
ndbcluster
datadir=/opt/mysqlcluster/deploy/mysqld_data
basedir=/opt/mysqlcluster/home/mysqlc
port=3306

[mysql_cluster]
ndb-connectstring=${MGMT_NODE_IP}
EOT

# MySQL cluster Management node config file
# 1 NDB_MGM node
# 3 replicas, or slaves
# MySQL nodes for master and data nodes
tee -a config.ini <<EOT
[ndb_mgmd]
hostname=${MGMT_NODE_IP}
datadir=/opt/mysqlcluster/deploy/ndb_data
nodeid=1

[ndbd default]
noofreplicas=3
datadir=/opt/mysqlcluster/deploy/ndb_data

[ndbd]
hostname=${DATA_NODE1_IP}
nodeid=3

[ndbd]
hostname=${DATA_NODE2_IP}
nodeid=4

[ndbd]
hostname=${DATA_NODE3_IP}
nodeid=5

[mysqld]
nodeid=50
EOT

# Setup base MySQL server, base MySQL tables and databases
cd /opt/mysqlcluster/home/mysqlc
sudo scripts/mysql_install_db --no-defaults --datadir=/opt/mysqlcluster/deploy/mysqld_data

# Starting the management node
sudo /opt/mysqlcluster/home/mysqlc/bin/ndb_mgmd -f /opt/mysqlcluster/deploy/conf/config.ini \
                                                --initial \
                                                --configdir=/opt/mysqlcluster/deploy/conf

# Cloning Sakila DB to be installed on the cluster
wget https://downloads.mysql.com/docs/sakila-db.tar.gz

tar -zxvf sakila-db.tar.gz
# Changing InnoDB engine to NDB engine
sed -i 's/InnoDB/ndb/g' /opt/mysqlcluster/home/mysqlc/sakila-db/sakila-schema.sql

# Starting MySQL server
/opt/mysqlcluster/home/mysqlc/bin/mysqld --defaults-file=/opt/mysqlcluster/deploy/conf/my.cnf --user=root &

sleep 60

# Setting up remote user on cluster
tee -a /home/ubuntu/setup_user.sql <<EOT
CREATE USER ${SQL_USER}@'%' IDENTIFIED BY '${SQL_PASSWORD}';
GRANT ALL PRIVILEGES ON *.* TO ${SQL_USER}@'%';
USE mysql;
DELETE FROM user WHERE user="" AND host="localhost";
FLUSH PRIVILEGES;
EOT

# Change root password
sudo /opt/mysqlcluster/home/mysqlc/bin/mysqladmin -u root password '${SQL_PASSWORD}'

# Install Sakila DB and remote user
mysql -u root -p${SQL_PASSWORD} < /opt/mysqlcluster/home/mysqlc/sakila-db/sakila-schema.sql
mysql -u root -p${SQL_PASSWORD} < /opt/mysqlcluster/home/mysqlc/sakila-db/sakila-data.sql
mysql -u root -p${SQL_PASSWORD} < /home/ubuntu/setup_user.sql

rm -f /home/ubuntu/setup_user.sql

# Installing sysbench script on server
apt-get install sysbench -y
tee -a /home/ubuntu/sysbench.sh <<EOT
#!/bin/bash

sudo mkdir /var/run/mysqld
sudo ln -s /tmp/mysql.sock /var/run/mysqld/mysqld.sock
sudo sysbench oltp_read_write --table-size=1000000 \
                              --db-driver=mysql \
                              --mysql-db=sakila \
                              --mysql-user=root \
                              --mysql-password=${SQL_PASSWORD} \
                              prepare
sudo sysbench oltp_read_write --threads=6 \
                              --time=60 \
                              --max-requests=0 \
                              --db-driver=mysql \
                              --mysql-db=sakila \
                              --mysql-user=root \
                              --mysql-password=${SQL_PASSWORD} \
                              run
sudo sysbench oltp_read_write --db-driver=mysql \
                              --mysql-db=sakila \
                              --mysql-user=root \
                              --mysql-password=${SQL_PASSWORD} \
                              cleanup
EOT

chmod +x /home/ubuntu/sysbench.sh
chown ubuntu:ubuntu /home/ubuntu/sysbench.sh