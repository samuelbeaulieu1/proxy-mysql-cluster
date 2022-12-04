#!/bin/bash

# If proxy PID was saved, kill the process
if [[ -f proxy.pid ]]; then
    PROXY_PID=$(cat proxy.pid)
    kill -9 $PROXY_PID
    rm -f proxy.pid
fi