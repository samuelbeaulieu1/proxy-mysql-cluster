#!/bin/bash

BUILD=false
STOP=false
MODE="MasterMode"
PORT="3306"

while getopts sbm:p: flag;
do
    case $flag in
        s|stop)     STOP=true       ;;
        b|build)    BUILD=true      ;;
        m|mode)     MODE="$OPTARG"  ;;
        p|port)     PORT=$OPTARG    ;;
    esac
done

set -a 
source .env
set +a

./stop_proxy.bash

if [ $BUILD = true ]; then
    echo "Building proxy"
    cd src
    sudo go build -v -o ../proxy
    cd ..
fi

if [ $STOP = false ]; then 
    ./proxy ${MODE} ${PORT} &
    PROXY_PID=$!

    echo $PROXY_PID > proxy.pid
fi