#!/bin/bash

if [[ -f proxy.pid ]]; then
    PROXY_PID=$(cat proxy.pid)
    kill -9 $PROXY_PID
    rm -f proxy.pid
fi