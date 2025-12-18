#!/bin/sh

while true; do
    socat TCP-LISTEN:31337,reuseaddr,fork,keepalive SYSTEM:"timeout -s SIGKILL 30 ./archive"
done
