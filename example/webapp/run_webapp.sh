#!/bin/bash

set -e

GO_VERSION=$(go version | awk '{print $3}' | cut -c 3-)
MIN_VERSION="1.23"

if [ "$(printf "$GO_VERSION\n$MIN_VERSION" | sort -V | head -n1)" = "$MIN_VERSION" ]; then
    GO_BUILD_LDFLAGS="-checklinkname=0"
else
    GO_BUILD_LDFLAGS=""
fi

echo "build webapp..."
go build -gcflags=all=-l -ldflags="-X main.HotfixVersion=main ${GO_BUILD_LDFLAGS}" -o webapp .

echo "please modify v1 plugin, press enter key to continue..."
read input

echo "build webapp plugin v1..."
go build -gcflags=all=-l -buildmode=plugin -ldflags="-X main.HotfixVersion=v1 ${GO_BUILD_LDFLAGS}" -o webapp_v1.so .

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