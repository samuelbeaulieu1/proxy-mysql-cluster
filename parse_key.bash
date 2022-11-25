#!/bin/bash

cd infrastructure/terraform

key=$(terraform show -json | jq | grep "private_key_openssh" | awk 'BEGIN {FS = ":"} ; {print $2}' | sed 's/\"//g; s/\,//g')

cd ../..

echo $key > key.pem

sed -i 's/\\n/\
/g' key.pem
sudo chmod 600 key.pem