#!/bin/bash
# SermonROAD 맥 빌드 (맥에서만 돌아요 — 맥 빌드 서버 scripts/ci-mac.sh 가 불러요)
#   Go 엔진(darwin/arm64) + Swift 앱 껍데기(app/shell/main.swift) → dist/SermonROAD.app
#   → 앱 전체에 임시 서명(ad-hoc) → dist/SermonROAD-<버전>.zip (업데이트용) · dist/SermonROAD.zip (처음 설치용, 사용 가이드 포함)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$(tr -d ' \n' < "$ROOT/VERSION")"
DIST="$ROOT/dist"
APP="$DIST/SermonROAD.app"
rm -rf "$DIST"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

echo "▶ 엔진 테스트 · 빌드 (darwin/arm64, v$VERSION)"
( cd "$ROOT/engine" && go test ./... )
( cd "$ROOT/engine" && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
    go build -trimpath -ldflags "-s -w -X main.build=$VERSION" -o "$APP/Contents/Resources/sermonroad-engine" . )
chmod 755 "$APP/Contents/Resources/sermonroad-engine"

echo "▶ 앱 껍데기 빌드 (Swift)"
xcrun swiftc --version
xcrun swiftc -O -target arm64-apple-macos12.0 \
  -o "$APP/Contents/MacOS/SermonROAD" "$ROOT/app/shell/main.swift" \
  -framework AppKit -framework WebKit -framework PDFKit -framework Vision
chmod 755 "$APP/Contents/MacOS/SermonROAD"

sed "s/__VERSION__/$VERSION/g" "$ROOT/app/Info.plist" > "$APP/Contents/Info.plist"
plutil -lint "$APP/Contents/Info.plist"
cp "$ROOT/app/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"
printf '%s' "$VERSION" > "$APP/Contents/Resources/version.txt"
printf 'APPL????' > "$APP/Contents/PkgInfo"

echo "▶ 임시 서명 (ad-hoc) · 확인"
codesign --force --deep --sign - --timestamp=none "$APP"
codesign --verify --deep --strict --verbose=2 "$APP"
codesign -dv "$APP" 2>&1 | sed 's/^/  /'

echo "▶ 압축"
ditto -c -k --norsrc --keepParent "$APP" "$DIST/SermonROAD-$VERSION.zip"
mkdir -p "$DIST/first/SermonROAD"
ditto "$APP" "$DIST/first/SermonROAD/SermonROAD.app"
[ -f "$ROOT/docs/SermonROAD_사용가이드.png" ] && cp "$ROOT/docs/SermonROAD_사용가이드.png" "$DIST/first/SermonROAD/"
ditto -c -k --norsrc --keepParent "$DIST/first/SermonROAD" "$DIST/SermonROAD.zip"

echo "▶ 업데이트 파일을 풀어도 서명이 그대로인지 확인 (앱의 업데이트 기능처럼 unzip)"
CHECK="$DIST/check"; rm -rf "$CHECK"; mkdir -p "$CHECK"
( cd "$CHECK" && unzip -q "$DIST/SermonROAD-$VERSION.zip" )
codesign --verify --deep --strict "$CHECK/SermonROAD.app"
[ "$(cat "$CHECK/SermonROAD.app/Contents/Resources/version.txt")" = "$VERSION" ]
rm -rf "$CHECK"
echo "✔ dist/SermonROAD-$VERSION.zip · dist/SermonROAD.zip"
