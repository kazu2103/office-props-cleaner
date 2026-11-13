#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

APP="dist/OfficePropsCleaner.app"
uv run --no-project --python-preference only-managed --python 3.13 --with pyinstaller \
  pyinstaller --noconfirm --windowed --name "OfficePropsCleaner" app.py

# PyInstaller が Tcl/Tk 9 のライブラリ末尾に付いたスクリプト群を落としてしまい起動できないため、元のファイルで上書きして署名し直す
PY_LIB="$(dirname "$(dirname "$(uv python find --python-preference only-managed 3.13)")")/lib"
for lib in libtcl9.0.dylib libtcl9tk9.0.dylib; do
  cp "$PY_LIB/$lib" "$APP/Contents/Frameworks/$lib"
  codesign --force -s - "$APP/Contents/Frameworks/$lib"
done
codesign --force -s - "$APP"
codesign --verify --deep --strict "$APP"
echo "built: $APP"
