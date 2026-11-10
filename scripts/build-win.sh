#!/bin/bash
# SermonROAD Windows 빌드 — 리눅스에서 바로 만들어요 (맥 빌드 서버 필요 없음)
#   SermonROAD.exe 하나 = 실행기 + 엔진 + 서재 PDF 읽기 도구 (engine/launcher_windows.go · pdftext_pdfium.go)
#   아이콘 · 매니페스트는 go-winres 로 넣어요.
#   → releases/SermonROAD-win-<v>.zip (업데이트용: SermonROAD.exe + version.txt) · releases/latest-win.json
#   → download/SermonROAD-Windows.zip (처음 설치용: SermonROAD 폴더 안에 SermonROAD.exe + 사용 가이드)
#
#   scripts/build-win.sh --notes "바뀐 점 1" "바뀐 점 2"
#   scripts/build-win.sh --test        (exe 만 dist/win/stage 에 만들고 releases 는 그대로)
# 필요: Go 1.24+, python3, go-winres (go install github.com/tc-hib/go-winres@v0.3.3)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$(tr -d ' \n' < "$ROOT/VERSION")"
NOTES=(); TEST=0
if [ "${1:-}" = "--notes" ]; then shift; NOTES=("$@"); fi
if [ "${1:-}" = "--test" ]; then TEST=1; fi
WIN="$ROOT/dist/win"
rm -rf "$WIN"; mkdir -p "$WIN/stage" "$WIN/SermonROAD" "$ROOT/releases"

echo "▶ 엔진 테스트 · 빌드 (windows/amd64, v$VERSION)"
( cd "$ROOT/engine" && go test ./... )
WINRES="${GO_WINRES:-$(go env GOPATH)/bin/go-winres}"
[ -x "$WINRES" ] || go install github.com/tc-hib/go-winres@v0.3.3
( cd "$ROOT/engine" && "$WINRES" simply --arch amd64 --manifest gui --icon "$ROOT/app/AppIcon.png" \
    --product-name SermonROAD --file-description "SermonROAD - sermon records and preparation" \
    --product-version "$VERSION.0" --file-version "$VERSION.0" --original-filename SermonROAD.exe \
    --copyright "made by. Nevertheless_D" --out rsrc )
trap 'rm -f "$ROOT/engine/rsrc_windows_amd64.syso"' EXIT
( cd "$ROOT/engine" && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w -H windowsgui -X main.build=$VERSION" -o "$WIN/stage/SermonROAD.exe" . )
printf '%s' "$VERSION" > "$WIN/stage/version.txt"
echo "✔ dist/win/stage/SermonROAD.exe ($(du -h "$WIN/stage/SermonROAD.exe" | cut -f1))"
[ "$TEST" = 1 ] && exit 0

ZIP="SermonROAD-win-$VERSION.zip"
python3 "$ROOT/scripts/zipdir.py" "$WIN/stage" "$ROOT/releases/$ZIP"
python3 "$ROOT/scripts/feed.py" "$ROOT/releases/latest-win.json" "$VERSION" "$ROOT/releases/$ZIP" "${NOTES[@]+"${NOTES[@]}"}"
find "$ROOT/releases" -maxdepth 1 -name "SermonROAD-win-*.zip" ! -name "$ZIP" -delete   # 업데이트용 zip 은 최신판만
bash "$ROOT/scripts/publish-download.sh"   # download/SermonROAD-Windows.zip 도 새 버전으로 (커밋 · 푸시)
