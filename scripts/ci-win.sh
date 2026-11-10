#!/bin/bash
# Windows 빌드 서버에서 돌아요 (Git Bash): 빌드 → 실제 Windows 시험 → 결과를 ci-results-win 브랜치에 올림 (실패해도 기록은 올려요)
set -uo pipefail
ROOT="$(pwd)"
OUT="$ROOT/ci-out"; rm -rf "$OUT"; mkdir -p "$OUT"
VERSION="$(tr -d ' \n' < VERSION)"
{ echo "commit: ${GITHUB_SHA:-local}  version: $VERSION  started: $(date -u +%FT%TZ)"; cmd //c ver; } > "$OUT/info.txt" 2>&1

BUILD=fail; SMOKE=skip
if bash scripts/build-win.sh --test > "$OUT/build.log" 2>&1; then
  BUILD=ok
  export PYTHONUTF8=1 PYTHONIOENCODING=utf-8
  python scripts/smoke-win.py dist/win/stage/SermonROAD.exe "$OUT/smoke" > "$OUT/smoke.log" 2>&1
  SMOKE="$(python -c "import json;print(json.load(open('$OUT/smoke/status.json'))['smoke'])" 2>/dev/null || echo fail)"
fi
printf '{"version":"%s","commit":"%s","build":"%s","smoke":"%s"}\n' "$VERSION" "${GITHUB_SHA:-local}" "$BUILD" "$SMOKE" > "$OUT/status.json"
cat "$OUT/status.json"; tail -30 "$OUT/build.log"; [ -f "$OUT/smoke.log" ] && cat "$OUT/smoke.log"

if [ -n "${GITHUB_TOKEN:-}" ]; then
  cd "$OUT"
  git init -q && git checkout -q -b ci-results-win
  git add -A
  git -c user.name="SermonROAD build" -c user.email="NeverthelessD@users.noreply.github.com" commit -qm "v$VERSION win build=$BUILD smoke=$SMOKE (${GITHUB_SHA:-local})"
  git push -qf "https://x-access-token:${GITHUB_TOKEN}@github.com/${GITHUB_REPOSITORY}.git" ci-results-win
fi
[ "$BUILD" = ok ] && [ "$SMOKE" = ok ]
