#!/bin/bash

sudo apt-get update -y \
    && sudo apt-get install -y git \
                               mariadb-client

cd /home/ubuntu

# Cloning repository to get proxy application
git clone https://github.com/samuelbeaulieu1/proxy-mysql-cluster.git

# Installing Golang
wget  https://go.dev/dl/go1.19.linux-amd64.tar.gz 
sudo tar -xvf go1.19.linux-amd64.tar.gz
sudo mv go /usr/local

sudo ln -s /usr/local/go/bin/go /usr/bin/go

# Setting up dependancies
cd proxy-mysql-cluster/app/src
sudo go mod tidy
cd ..

# environment variables for Proxy about MySQL cluster setup
tee -a .env <<EOT
SQL_PASSWORD="${SQL_PASSWORD}"
SQL_USER="${SQL_USER}"
MASTER_NODE_IP="${MASTER_NODE_IP}"
N_SLAVES=${N_SLAVES}
DATA_NODE1_IP="${DATA_NODE1_IP}"
DATA_NODE2_IP="${DATA_NODE2_IP}"
DATA_NODE3_IP="${DATA_NODE3_IP}"
EOT

# Starting the proxy with port 3306 and Direct hit mode by default
./run.bash -b -p 3306 -m MasterMode