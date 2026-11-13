#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

OUT=../dist/mac
APP="$OUT/OfficePropsCleaner.app"
rm -rf "$APP" build
mkdir -p build "$APP/Contents/MacOS"

MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS="-mmacosx-version-min=13.0" CGO_LDFLAGS="-mmacosx-version-min=13.0" CGO_ENABLED=1 go build -trimpath -buildmode=c-archive -o build/libbridge.a ./bridge
swiftc -O -target arm64-apple-macos13.0 -import-objc-header build/libbridge.h App.swift build/libbridge.a \
  -framework AppKit -framework UniformTypeIdentifiers \
  -o "$APP/Contents/MacOS/OfficePropsCleaner"

cat > "$APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>OfficePropsCleaner</string>
<key>CFBundleIdentifier</key><string>io.github.kazu2103.office-props-cleaner</string>
<key>CFBundleName</key><string>OfficePropsCleaner</string>
<key>CFBundleDisplayName</key><string>Officeプロパティ消去</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>1.0</string>
<key>LSMinimumSystemVersion</key><string>13.0</string>
<key>LSUIElement</key><true/>
</dict></plist>
PLIST

codesign --force -s - "$APP"
codesign --verify --strict "$APP"
echo "built: $APP"
