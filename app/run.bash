#!/bin/bash

# Compile the proxy
BUILD=false
# Stop the proxy
STOP=false
# Proxy mode (MasterMode | RandomMode | LatencyMode)
MODE="MasterMode"
# Proxy listen port, default 3306 for MySQL default port
PORT="3306"

# Input arguments to script
while getopts sbm:p: flag;
do
    case $flag in
        s|stop)     STOP=true       ;;
        b|build)    BUILD=true      ;;
        m|mode)     MODE="$OPTARG"  ;;
        p|port)     PORT=$OPTARG    ;;
    esac
done

# Setting env vars from .env file
set -a 
source .env
set +a

# Stop current proxy, if currently running
./stop_proxy.bash

# If -b was sent in arguments, compile the proxy
if [ $BUILD = true ]; then
    echo "Building proxy"
    cd src
    sudo go build -v -o ../proxy
    cd ..
fi

# If -s was not set in arguments, restart the proxy with the mode and port
if [ $STOP = false ]; then 
    ./proxy ${MODE} ${PORT} &
    PROXY_PID=$!

    # Save proxy PID to be able to stop later
    echo $PROXY_PID > proxy.pid
fi