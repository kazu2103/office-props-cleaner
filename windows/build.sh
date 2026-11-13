#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H=windowsgui -s -w" -o dist/windows/OfficePropsCleaner.exe ./windows
echo "built: dist/windows/OfficePropsCleaner.exe"
