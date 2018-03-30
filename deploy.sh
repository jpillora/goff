#!/bin/bash
echo "building" &&
GOOS=linux go build -ldflags "-w -s -X main.BuiltTime=$(date -u +%s)" -o /tmp/gobin &&
# echo "upxing" &&
# upx /tmp/gobin &&
echo "uploading" &&
rsync -e "ssh -p 6969" --compress /tmp/gobin root@vultr.jpillora.com:/usr/local/bin/goff &&
echo "done" &&
rm /tmp/gobin
