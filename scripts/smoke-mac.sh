#!/bin/bash
# SermonROAD 실제 맥 시험 (맥 빌드 서버에서 dist/SermonROAD.app 을 /Applications 에 깔고 여러 상황을 시험해요)
# 결과: $OUT/results.txt (PASS/FAIL/INFO 한 줄씩), 스크린샷, 기록 파일
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${OUT:-$ROOT/ci-out}"
mkdir -p "$OUT"
APPDIR="/Applications/SermonROAD.app"
SUP="$HOME/Library/Application Support/SermonROAD"
RES="$OUT/results.txt"
: > "$RES"

pass() { echo "PASS  $*" | tee -a "$RES"; }
fail() { echo "FAIL  $*" | tee -a "$RES"; }
info() { echo "INFO  $*" | tee -a "$RES"; }
check() { local name="$1"; shift; if "$@"; then pass "$name"; else fail "$name"; fi; }

app_pid() { pgrep -x SermonROAD | head -1; }
engines() { pgrep -x sermonroad-engine | tr '\n' ' '; }
tok() { cat "$SUP/token" 2>/dev/null; }
port() { cat "$SUP/port" 2>/dev/null; }
API() { echo "http://127.0.0.1:$(port)"; }
get() { curl -fs -m 10 -H "x-token: $(tok)" "$(API)$1"; }
post() { curl -fs -m 30 -X POST -H "x-token: $(tok)" -H 'content-type: application/json' --data "$2" "$(API)$1"; }
ping_ok() { get /api/ping >/dev/null 2>&1; }
wait_ready() { for _ in $(seq 1 60); do [ -s "$SUP/port" ] && ping_ok && return 0; sleep 0.5; done; return 1; }
wait_gone() { for _ in $(seq 1 30); do [ -z "$(app_pid)" ] && [ -z "$(engines)" ] && return 0; sleep 0.5; done; return 1; }
snap() { echo "--- $1"; ps -axo pid,ppid,stat,comm | grep -E "SermonROAD|sermonroad-engine" | grep -v grep; } >> "$OUT/procs.txt"
launch() { open "$APPDIR"; wait_ready; }
jq_() { python3 -c "import json,sys; d=json.load(sys.stdin); print($1)" 2>/dev/null; }

echo "▶ 준비: 앱 설치 · 설정"
rm -rf "$APPDIR"; ditto "$ROOT/dist/SermonROAD.app" "$APPDIR"
rm -rf "$SUP"; mkdir -p "$SUP"
printf '{"appearance":"light","autoUpdate":false,"claudeMode":"api","showBoth":true,"model":"claude-sonnet-5-5","standards":["wcf","wsc","hc","cd"],"ui":{"homePlace":"본 교회"}}' > "$SUP/settings.json"

echo "▶ A. 켜기 · 화면 · 기록"
rm -rf "$SUP/backup.noindex"; mkdir -p "$SUP/backup"; ditto "$APPDIR" "$SUP/backup/SermonROAD.app"
printf '0.9.0' > "$SUP/backup/SermonROAD.app/Contents/Resources/version.txt"
launch; check "A1 앱을 열면 엔진이 준비됨" ping_ok
check "A1b 이전 버전 보관본을 Spotlight 가 안 찾는 폴더(backup.noindex)로 옮김" bash -c "test -d '$SUP/backup.noindex/SermonROAD.app' && test ! -e '$SUP/backup'"
check "A2 엔진이 Dock 앱 방식으로 켜짐(shell=native)" bash -c "curl -fs -H 'x-token: $(tok)' $(API)/api/ping | grep -q '\"shell\":\"native\"'"
check "A3 주소가 47931" test "$(port)" = "47931"
SER=$(post /api/series/save '{"series":{"name":"로마서 강해","kind":"주일","total":52,"nextRef":"롬 8:31-39","color":"#9B6BF2"}}' | jq_ "d['series']['id']")
R1=$(post /api/sermon/save "{\"sermon\":{\"date\":\"2026-11-01\",\"time\":\"11:00\",\"kind\":\"주일\",\"title\":\"모든 것이 합력하여 선을 이루느니라\",\"passageText\":\"로마서 8장 28~30절\",\"place\":\"본 교회\",\"status\":\"설교함\",\"seriesId\":\"$SER\",\"seriesNo\":23,\"points\":[\"합력하여 선을 이루시는 하나님\",\"끊어지지 않는 구원의 사슬\"],\"confessions\":[{\"doc\":\"wcf\",\"n\":3}]}}")
REF=$(echo "$R1" | jq_ "d['sermon']['passages'][0]['ref']")
check "A4 설교를 저장하고 본문을 알아봄 (${REF:-?})" test "$REF" = "롬 8:28–30"
for d in 2025-03-02 2024-06-09 2023-05-14; do
  post /api/sermon/save "{\"sermon\":{\"date\":\"$d\",\"kind\":\"주일\",\"title\":\"시험 설교 $d\",\"passageText\":\"요 3:16\",\"place\":\"본 교회\",\"status\":\"설교함\"}}" > /dev/null
done
post /api/rule/save "{\"rule\":{\"dow\":0,\"every\":1,\"time\":\"11:00\",\"kind\":\"주일\",\"place\":\"본 교회\",\"seriesId\":\"$SER\",\"active\":true}}" > /dev/null
N=$(get /api/data | jq_ "len(d['sermons'])")
check "A5 기록을 다시 읽음 (설교 ${N:-?}편)" test "${N:-0}" = "4"
check "A6 data.json 저장 · 오늘 자동 백업" bash -c "test -s '$SUP/data.json' && ls '$SUP/backups/' | grep -q '^auto-'"
sleep 6
screencapture -x "$OUT/shot-app.png"
check "A7 화면이 앱 창(WKWebView)에서 열림" grep -q "SermonROADShell" "$SUP/requests.log"
APID=$(app_pid); snap "A 실행 중"
SUPPID=$(ps -axo pid=,ppid=,comm= | awk -v a="$APID" '$2==a && /sermonroad-engine/ {print $1}' | head -1)
CHILD=$(ps -axo pid=,ppid=,comm= | awk -v s="$SUPPID" '$2==s && /sermonroad-engine/ {print $1}' | head -1)
check "A8 엔진이 SermonROAD 앱의 자식 (앱 $APID → 지킴이 ${SUPPID:-없음} → 엔진 ${CHILD:-없음})" test -n "$SUPPID" -a -n "$CHILD"

echo "▶ L. 서재: PDF 글자 읽기 · 글자 인식(스캔) · 본문으로 찾기"
LIB="$HOME/SermonROAD-test-library"; rm -rf "$LIB"; mkdir -p "$LIB"; cp "$ROOT"/scripts/testdata/*.pdf "$LIB/"
recents() { defaults read com.apple.dock recent-apps 2>/dev/null | grep -c SermonROAD; }
REC0=$(recents)
"$APPDIR/Contents/Resources/sermonroad-pdftext" "$LIB/스캔 시험.pdf" "$OUT/scan.json" > "$OUT/pdf-text.log" 2>&1; PRC=$?
check "L1 앱 안의 도구가 PDF 글자를 읽음 (코드 $PRC)" test "$PRC" = "0" -a -s "$OUT/scan.json"
OCRT=$(python3 -c "import json; d=json.load(open('$OUT/scan.json')); print(len(d['ocr']), d['pages'][0][:60].replace(chr(10),' '))" 2>/dev/null)
info "글자 인식 결과: $OCRT"
check "L2 스캔 PDF 를 맥 글자 인식으로 읽음" bash -c "python3 -c \"import json,sys; d=json.load(open('$OUT/scan.json')); t=d['pages'][0]; sys.exit(0 if d['ocr'] and ('로마서' in t or '합력' in t or '8:28' in t) else 1)\""
# Dock 에 보이는(Foreground) SermonROAD 앱 수
apps() { lsappinfo list 2>/dev/null | awk '/"SermonROAD"/{f=1} f&&/type=/{if($0~/Foreground/)c++; f=0} END{print c+0}'; }
MAXAPPS=0
post /api/library/folder "{\"path\":\"$LIB\"}" > /dev/null
for _ in $(seq 1 240); do
  A=$(apps); [ "${A:-0}" -gt "$MAXAPPS" ] && MAXAPPS=$A
  R=$(get /api/library | jq_ "d['ready']"); [ "${R:-0}" = "3" ] && break; sleep 0.5
done
get /api/library > "$OUT/library.json"
check "L3 서재 PDF 3개를 모두 읽음 (읽음 ${R:-?})" test "${R:-0}" = "3"
check "L3b PDF 를 읽는 동안 Dock 에 앱이 하나만 (가장 많을 때 ${MAXAPPS}개)" test "$MAXAPPS" -le 1
sleep 1; REC1=$(recents)
check "L3c PDF 를 읽어도 Dock '최근 앱'에 SermonROAD 가 늘지 않음 ($REC0 → $REC1)" test "${REC1:-0}" -le "${REC0:-0}"
HITS=$(post /api/library/find '{"ref":"롬 8:28-30"}' | tee "$OUT/find.json" | jq_ "len(d['hits'])")
check "L4 서재에서 롬 8:28-30 을 다룬 쪽을 찾음 (${HITS:-0}곳)" test "${HITS:-0}" -ge 2

echo "▶ K. 키체인 (API 키 저장 · 읽기 · 지우기)"
KR=$(post /api/claude/key '{"key":"sk-ant-smoketest-0000000000000000000000"}' 2>&1 | jq_ "d['hasKey']")
if [ "$KR" = "True" ]; then
  pass "K1 API 키를 키체인에 저장하고 다시 읽음"
  post /api/claude/key '{"key":""}' > /dev/null
  KD=$(get /api/claude | jq_ "d['hasKey']")
  check "K2 API 키를 지움" test "$KD" = "False"
else
  info "K1 빌드 서버의 키체인을 쓸 수 없어서 건너뜀 (${KR:-응답 없음})"
fi

echo "▶ X. 엑셀로 내보내기 · 인쇄용 목록"
XP=$(post /api/export '{"scope":"all"}' | jq_ "d['path']")
check "X1 엑셀 파일을 만듦 (${XP:-없음})" bash -c "test -s \"$XP\" && unzip -l \"$XP\" | grep -q sheet1.xml"
check "X2 인쇄용 목록" bash -c "curl -fs '$(API)/print?t=$(tok)&scope=all' | grep -q '모든 것이 합력하여'"
osascript -e 'tell application "Finder" to close every window' >/dev/null 2>&1

echo "▶ B. Dock 아이콘 다시 누르기(reopen)"
osascript -e 'tell application "SermonROAD" to reopen' >/dev/null 2>&1; sleep 1
check "B1 다시 열기 뒤에도 앱·엔진 그대로" test "$(app_pid)" = "$APID"

echo "▶ C. 엔진이 고장으로 꺼지면 지킴이가 다시 켬"
kill -SEGV "$CHILD" 2>/dev/null; sleep 4
check "C1 엔진 다시 켜짐" ping_ok
check "C2 앱은 그대로" test "$(app_pid)" = "$APID"
N2=$(get /api/data | jq_ "len(d['sermons'])")
check "C3 다시 켜진 뒤에도 기록 그대로 (${N2:-?}편)" test "${N2:-0}" = "4"
sleep 2; screencapture -x "$OUT/shot-app-2.png"

echo "▶ D. 앱 종료(⌘Q) → 엔진까지 모두 꺼짐"
osascript -e 'tell application "SermonROAD" to quit' >/dev/null 2>&1
check "D1 앱·엔진 모두 꺼짐" wait_gone
snap "D 종료 뒤"
cp "$SUP/clean-exit" "$OUT/clean-exit-D.txt" 2>/dev/null

echo "▶ E. 화면의 '종료'(= /api/quit) → 앱도 꺼짐"
launch
post /api/quit '{}' >/dev/null
check "E1 앱·엔진 모두 꺼짐" wait_gone

echo "▶ F. 앱이 강제로 꺼지면 엔진도 스스로 꺼짐"
launch; kill -9 "$(app_pid)"; sleep 1
check "F1 남은 엔진 없음" bash -c 'for i in $(seq 1 20); do [ -z "$(pgrep -x sermonroad-engine)" ] && exit 0; sleep 0.5; done; exit 1'

echo "▶ G. 업데이트 뒤 다시 열기 (/api/update/restart)"
launch; OLD=$(app_pid)
post /api/update/restart '{}' >/dev/null
sleep 3
check "G1 새로 열린 앱이 엔진을 켬" wait_ready
NEW=$(app_pid)
check "G2 앱이 새로 열림 ($OLD → ${NEW:-없음})" test -n "$NEW" -a "$NEW" != "$OLD"
check "G3 앱 하나만 떠 있음 ($(pgrep -x SermonROAD | wc -l | tr -d ' ')개)" test "$(pgrep -x SermonROAD | wc -l | tr -d ' ')" = "1"
osascript -e 'tell application "SermonROAD" to quit' >/dev/null 2>&1
check "G4 끝나면 모두 꺼짐" wait_gone

echo "▶ 기록 모으기"
for f in launcher.log requests.log engine.log engine.prev.log engine.out engine.prev.out settings.json; do
  [ -f "$SUP/$f" ] && cp "$SUP/$f" "$OUT/$f"
done
sw_vers > "$OUT/macos.txt"
echo
echo "== 결과 =="; cat "$RES"
! grep -q '^FAIL' "$RES"
