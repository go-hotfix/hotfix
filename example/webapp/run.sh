#!/bin/bash

set -e

echo "=== Step 0: Reset ==="
git checkout -- service/discount.go 2>/dev/null || true
pkill -f './webapp' 2>/dev/null || true
rm -f webapp webapp_v1.so
sleep 1

echo ""
echo "=== Step 1: Build webapp ==="
go build -gcflags="all=-l -N" -o webapp .

echo ""
echo "=== Step 2: Start webapp ==="
./webapp &
WEBAPP_PID=$!
sleep 1

echo ""
echo "=== Step 3: Demo bug — VIP discount should be 20 but returns 0 ==="
echo "Request: GET /order?price=100&level=vip"
curl -s http://127.0.0.1:8080/order?price=100\&level=vip

echo ""
echo "=== Step 4: Apply fix (overwrite discount.go with fixed version) ==="
cp service/discount.go.fixed service/discount.go

echo ""
echo "=== Step 5: Build plugin from fixed source ==="
go build -gcflags="all=-l -N" -buildmode=plugin -o webapp_v1.so .

echo ""
echo "=== Step 6: Restore original (buggy) source ==="
git checkout -- service/discount.go 2>/dev/null || true

echo ""
echo "=== Step 7: Hotfix — load plugin and patch calcDiscount ==="
curl -s http://127.0.0.1:8080/hotfix

echo ""
echo "=== Step 8: Verify fix — VIP discount should now be 20 ==="
echo "Request: GET /order?price=100&level=vip"
curl -s http://127.0.0.1:8080/order?price=100\&level=vip

echo ""
echo "=== Step 9: Cleanup ==="
kill $WEBAPP_PID 2>/dev/null || true
rm -f webapp webapp_v1.so

echo ""
echo "=== Done ==="
