#!/bin/bash
echo "building" &&
  GOOS=linux GOARCH=amd64 go build -ldflags "-w -s -X main.built=$(date -u +%s)" -o /tmp/gobin &&
  # echo "upxing" &&
  # upx /tmp/gobin &&
  echo "uploading" &&
  rsync --compress /tmp/gobin kjp:/usr/local/bin/goff &&
  echo "done" &&
  rm /tmp/gobin
cd
