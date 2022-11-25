#!/bin/bash

sudo apt-get update -y \
    && sudo apt-get install -y git \
                               mariadb-client

cd /home/ubuntu

git clone https://github.com/samuelbeaulieu1/proxy-mysql-cluster.git

wget  https://go.dev/dl/go1.19.linux-amd64.tar.gz 
sudo tar -xvf go1.19.linux-amd64.tar.gz
sudo mv go /usr/local

sudo ln -s /usr/local/go/bin/go /usr/bin/go

cd proxy-mysql-cluster/app/src
sudo go mod tidy
cd ..

tee -a .env <<EOT
SQL_PASSWORD="${SQL_PASSWORD}"
SQL_USER="${SQL_USER}"
MASTER_NODE_IP="${MASTER_NODE_IP}"
N_SLAVES=${N_SLAVES}
DATA_NODE1_IP="${DATA_NODE1_IP}"
DATA_NODE2_IP="${DATA_NODE2_IP}"
DATA_NODE3_IP="${DATA_NODE3_IP}"
EOT

./run.bash -b -p 3306 -m MasterMode