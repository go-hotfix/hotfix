#!/bin/bash

set -e

echo "build webapp..."
go build -gcflags=all=-l -ldflags="-X main.HotfixVersion=main" -o webapp .

echo "please modify v1 plugin, press enter key to continue..."
read input

echo "build webapp plugin v1..."
go build -gcflags=all=-l -buildmode=plugin -ldflags="-X main.HotfixVersion=v1" -o webapp_v1.so .

echo "run main program..."
./webapp &

# wait webapp to start
sleep 2s

echo "(before): get server response..."
curl http://127.0.0.1:8080/now

echo "(hotfix): start hotfix...."
curl http://127.0.0.1:8080/hotfix

echo "(after): get server response..."
curl http://127.0.0.1:8080/now

# kill webapp
pkill webapp