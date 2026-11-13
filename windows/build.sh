#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
# walk はコモンコントロール v6 を要求するので、マニフェストを .syso にして実行ファイルへ埋め込む
go run github.com/akavel/rsrc@v0.10.2 -manifest windows/app.manifest -ico windows/icon.ico -arch amd64 -o windows/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H=windowsgui -s -w" -o dist/windows/OfficePropsCleaner.exe ./windows
echo "built: dist/windows/OfficePropsCleaner.exe"
