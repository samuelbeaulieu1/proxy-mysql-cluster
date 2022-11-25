#!/bin/bash

DEPLOY=false
TEARDOWN=false
MYSQL_PASSWORD="123"

AWS_CREDENTIALS=~/.aws/credentials

TERRAFORM_IMAGE_TAG="hashicorp/terraform:1.3.2"

while getopts dtp: flag;
do
    case "${flag}" in
        d) DEPLOY=true;;
        t) TEARDOWN=true;;
        p) MYSQL_PASSWORD=${OPTARG};;
    esac
done

# Get credentials from ~/.aws/credentials if env vars are not set
if [[ -z $AWS_ACCESS_KEY_ID ]] || [[ -z $AWS_SECRET_ACCESS_KEY ]] || [[ -z $AWS_SESSION_TOKEN ]]; then
    values=$(awk -F '=' '{
        if ($1 != "[default]") {
            print $0;
        }
    }' $AWS_CREDENTIALS)

    for val in $values
    do
        IFS='='
        read -a name_with_value <<< $val
        if [[ ${name_with_value[0]} == "aws_access_key_id" ]]; then 
            AWS_ACCESS_KEY_ID=${name_with_value[1]} 
        fi 
        if [[ ${name_with_value[0]} == "aws_secret_access_key" ]]; then 
            AWS_SECRET_ACCESS_KEY=${name_with_value[1]} 
        fi 
        if [[ ${name_with_value[0]} == "aws_session_token" ]]; then 
            AWS_SESSION_TOKEN=${name_with_value[1]} 
        fi 
    done
fi

if [ $DEPLOY = true ]; then
    echo "Setting up AWS environment..."
    # Terraform init
    docker run \
        --env AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
        --env AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
        --env AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \
        --env AWS_REGION="us-east-1" \
        --env TF_VAR_sql_password=$MYSQL_PASSWORD \
        -v $(pwd)/infrastructure/:/root/infrastructure \
        -t $TERRAFORM_IMAGE_TAG \
        -chdir=/root/infrastructure/terraform/ init

    # Terraform apply
    docker run \
        --env AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
        --env AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
        --env AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \
        --env TF_VAR_sql_password=$MYSQL_PASSWORD \
        --env AWS_REGION="us-east-1" \
        -v $(pwd)/infrastructure/:/root/infrastructure \
        -v $(pwd)/app/:/root/app \
        -t $TERRAFORM_IMAGE_TAG \
        -chdir=/root/infrastructure/terraform/ apply -auto-approve

    ./parse_key.bash
fi

if [ $TEARDOWN = true ]; then
    echo "Tearing down AWS environment..."
    # Terraform destroy
    docker run \
        --env AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
        --env AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
        --env AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \
        --env AWS_REGION="us-east-1" \
        -v $(pwd)/infrastructure/:/root/infrastructure \
        -t $TERRAFORM_IMAGE_TAG \
        -chdir=/root/infrastructure/terraform/ apply -destroy -auto-approve
fi