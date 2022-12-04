#!/bin/bash

# Installing dependencies and docker
sudo apt-get update
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Getting Sakila DB
wget https://downloads.mysql.com/docs/sakila-db.tar.gz

tar -zxvf sakila-db.tar.gz

# Setting up MySQL entrypoint scripts to setup Sakila
sudo mv sakila-db/sakila-schema.sql sakila-db/01-sakila-schema.sql 
sudo mv sakila-db/sakila-data.sql sakila-db/02-sakila-data.sql 

# Starting MySQL stand-alone server
sudo docker run --name mysql-standalone \
                --env MYSQL_ROOT_PASSWORD=${SQL_PASSWORD} \
                -v $(pwd)/sakila-db:/docker-entrypoint-initdb.d \
                -p 3306:3306 \
                -d mysql:8.0.31

# Setting up sysbench user
tee -a /home/ubuntu/setup_sysbench.sql <<EOT
CREATE USER sbtest@'%' IDENTIFIED BY '${SQL_PASSWORD}';
GRANT ALL PRIVILEGES ON sakila.* TO sbtest@'%';
FLUSH PRIVILEGES;
EOT

# Installing sysbench script on server
apt-get install sysbench -y
tee -a /home/ubuntu/sysbench.sh <<EOT
#!/bin/bash

sudo docker exec -i mysql-standalone mysql -u root -p${SQL_PASSWORD} < /home/ubuntu/setup_sysbench.sql

CONTAINER_IP=$(sudo docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' mysql-standalone 2>/dev/null)

sudo sysbench oltp_read_write --table-size=1000000 \
                              --db-driver=mysql \
                              --mysql-db=sakila \
                              --mysql-user=sbtest \
                              --mysql-password=${SQL_PASSWORD} \
                              --mysql-host=\$CONTAINER_IP \
                              prepare
sudo sysbench oltp_read_write --threads=6 \
                              --time=60 \
                              --max-requests=0 \
                              --db-driver=mysql \
                              --mysql-db=sakila \
                              --mysql-user=sbtest \
                              --mysql-password=${SQL_PASSWORD} \
                              --mysql-host=\$CONTAINER_IP \
                              run
sudo sysbench oltp_read_write --db-driver=mysql \
                              --mysql-db=sakila \
                              --mysql-user=sbtest \
                              --mysql-password=${SQL_PASSWORD} \
                              --mysql-host=\$CONTAINER_IP \
                              cleanup
EOT

chmod +x /home/ubuntu/sysbench.sh
chown ubuntu:ubuntu /home/ubuntu/sysbench.sh