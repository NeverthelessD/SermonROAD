> **이 파일을 읽은 Claude는 작업이 끝날 때마다 이 파일을 업데이트할 것.**
> (구조가 바뀌면 「구조」, 새로 정한 것은 「결정 사항」, 한 일·남은 일은 「진행 사항」에 적고 커밋·푸시까지 할 것)

# SermonROAD — 프로젝트 노트

개혁주의 설교자를 위한 설교 기록 · 준비 앱. macOS (Apple Silicon) Dock 앱. made by. Nevertheless_D
사용자는 개발자가 아니에요. 한국어로 쉽게 설명하고, 터미널 사용을 요구하지 말 것. 결과물은 더블클릭으로 쓸 수 있게.
만드는 원칙은 road-mac-app 스킬과 MSROAD 저장소(CLAUDE.md · PROJECT_NOTES.md)를 따른다.

---

## 진행 사항

작업이 중간에 끊기면 아래 체크리스트에서 **처음으로 체크 안 된 항목부터** 이어서 하면 돼요.

- [x] 0. 요구 정리 (2026-11-04)
- [x] 1. 디자인 시안 확정 (2026-11-04 17:12, "이대로 진행, 아이콘은 D")
      시안: Design 캔버스 'SermonROAD 디자인 시안' (claude.ai/artifact/N6naHkxm5iUY47TVanoVYt)
- [x] 2. 저장소 뼈대 (CLAUDE.md, VERSION, .gitignore, app/shell, engine, scripts, CI)
- [x] 3. 엔진(Go): data.json 저장 · 자동 백업 · 휴지통, 본문 해석(bible.go, 시험 있음), 첨부, 엑셀 · 인쇄용 목록
- [x] 4. 서재: PDF 글자 읽기(맥: sermonroad-pdftext = PDFKit + Vision OCR, 리눅스: pdftotext), 본문 표기 색인, 찾기
- [x] 5. 설교 도우미: API(키체인) / 구독 복사 방식, 출처 칩 · 자료 밖 표시, 설교 기록에 저장, 개요 → 중심 명제 · 대지
- [x] 6. 화면 9개 (리눅스 Playwright 로 확인, 가짜 Claude 서버로 도우미 흐름 확인)
- [x] 7. 사용 가이드 이미지 (docs/SermonROAD_사용가이드.png ← docs/guide.html, scripts/guide.py)
- [ ] 8. 맥 빌드 서버 build=ok · smoke=ok → scripts/release.sh → download/SermonROAD.zip · 사용자에게 알리기
- [ ] 9. 사용자 맥에서 확인: 실제 서재 PDF(스캔본 포함) 글자 인식 속도 · 정확도, 실제 API 키로 도우미, 키체인 저장

## 구조

    app/shell/main.swift   Dock 앱 껍데기 (WKWebView, 엔진 켜고 끄기, 업데이트 뒤 다시 열기, 복사 · 폴더 고르기 다리)
    app/pdftext/main.swift 서재 PDF 글자 읽기 도구 → Contents/Resources/sermonroad-pdftext (PDFKit + Vision, 앱이 아니라 Dock 에 안 나타남)
    app/Info.plist · AppIcon.svg(아이콘 D) → scripts/icon.py → AppIcon.png · AppIcon.icns · engine/web/icon.png
    engine/main.go         설정 · 경로(127.0.0.1:47931 + 토큰) · API 목록
    engine/store.go        data.json (설교 · 시리즈 · 예화 · 반복 일정 · 휴지통), 백업(Support/backups), 첨부(Support/files/<설교>/)
    engine/bible.go        66권 표(HOLYBIBLE 코드 OLD01~39 · NEW01~27, 확인함) · 본문 읽기 · 겹침 · PDF 글 속 본문 표기 찾기
    engine/library.go      서재 폴더 훑기 · 글자 캐시(Support/library/<id>.json) · 본문으로 찾기
    engine/helper.go       도우미 요청문 · Claude API(tool 결과) · 키체인(/usr/bin/security) · 이번 달 사용량
    engine/export.go       엑셀(.xlsx 직접 만듦, ~/Downloads) · /print 인쇄용 목록(브라우저)
    engine/supervise.go · update.go · diag.go · proc_unix.go   MSROAD 와 같은 지킴이 · 업데이트 · 진단
    engine/web/            index.html · css/app.css(토큰) · css/gowun.css(고운바탕 OFL)
                           js/core.js(api · 상태 S · 절기 계산 · 모달 · 길 찾기) data.js(요리문답 · 신앙고백 · 색)
                           map · calendar · records · sermon · series · confess · illus · helper · stats · settings · main
    scripts/               build-mac · ci-mac · smoke-mac · release · publish-download · feed · icon · guide · make-test-pdfs
    scripts/testdata/      시험용 서재 PDF (글자 PDF 2개 + 스캔 PDF 1개)

## 결정 사항 (사용자 요청)

- 앱 이름 **SermonROAD**, 저장소 NeverthelessD/SermonROAD. 아이콘 **D '말씀 위의 펜촉'** (펼친 성경 위 펜촉, 펜촉 십자가 줄기 = 성경 가운데 선).
- 디자인(확정): 깊은 남색 + 금색(다크 #E2BE62 / 라이트 #8A6216), 성경 묶음 10색, Pretendard 본문 · Unbounded 로고 · 고운바탕(본문 · 큰 제목).
  메뉴: 설교 지도 · 설교 달력 · 설교 기록 · 강해 시리즈 · 요리문답 · 신앙고백 · 예화 보관함 · 설교 도우미 · 통계 · 백업 · 설정.
  화면 아래 `made by. Nevertheless_D`.
- '성경 66권 설교 지도' → **'설교 지도'**, 구약/신약 '다섯 길' 문구 없음.
- 설교 기록은 **기간 › 연도 › 월** 3단계 (2013–15, 16–20, 21–25, 26–30, 31–35, 36–40; 2013 이전 · 2040 이후 기록이 생기면 기간이 저절로 늘어요).
- 설교 달력(스케줄러): 월별, 종류 거르기, 준비 상태 4가지(본문 미정 · 준비 중 · 원고 완료 · 설교함), 절기 자동 표시,
  반복 일정(매주/격주, 시리즈 '다음 본문'을 가장 가까운 예정 칸에), 사흘 안 준비 늦은 설교 알림.
  절기: 종교개혁주일 = 10월 마지막 주일, 추수감사주일 = 11월 셋째 주일, 맥추감사주일 = 7월 첫 주일, 세계성찬주일 = 10월 첫 주일, 부활절 계산.
- 종류: 주일, 주일 오후, 새벽, 수요, 금요, 성찬, 그 외(직접 적기).
- 성경 본문은 직접 넣기(번역본 저작권 때문에 앱에 성경 본문을 넣지 않음) + **HOLYBIBLE**(https://www.biblenote.kr/read.html?book=NEW06&chapter=8) 링크. 절 단위 주소는 없음.
- 설교 도우미: 서재 PDF 에서 고른 본문을 다룬 쪽을 찾아 그 자료를 근거로 정리·요약 / 구조 / 묵상 / 설교 개요 / 개혁주의 관점.
  결과마다 출처(자료 · PDF 쪽), 자료에 없는 내용은 '자료 밖'. Claude 연결은 설정에서 고름(api | copy), 다른 방법 버튼도 보이기(기본 켬).
  모델 기본 claude-sonnet-5-5 (opus-5-5, haiku-5-5 고를 수 있음). 쪽 번호는 PDF 쪽 순서.
- 기록은 이 맥 안(~/Library/Application Support/SermonROAD/data.json)에만. 바뀐 날마다 자동 백업(최근 20) + 직접 백업(20) + 되살리기 전 백업.
- 커밋 작성자: `Nevertheless_D <NeverthelessD@users.noreply.github.com>`

## 시험하는 법 (리눅스)

    cd engine && go test ./... && go build -o /tmp/sre .
    SERMONROAD_SUPPORT=/tmp/sup SERMONROAD_TOKEN=testtoken SERMONROAD_EXPORT_DIR=/tmp \
      SERMONROAD_UPDATE_URL=http://127.0.0.1:1/none /tmp/sre     # → http://127.0.0.1:47931/?t=testtoken
    가짜 Claude: SERMONROAD_CLAUDE_URL=http://127.0.0.1:47999/v1/messages SERMONROAD_CLAUDE_KEY=sk-ant-test-…
    (화면 파일은 엔진에 들어가니 web/ 을 고치면 다시 빌드할 것)

## 업데이트 내보내는 법 (Claude용)

1. 수정 → `VERSION` 올리기 → 커밋 · 푸시 (맥 빌드 서버가 자동으로 돎, 5~10분)
2. `git ls-remote origin ci-results` 가 바뀔 때까지 기다린 뒤 결과 확인 (status.json, results.txt, shot-app.png)
3. `scripts/release.sh --notes "바뀐 점 1" "바뀐 점 2"` → 커밋 · 푸시 (download/SermonROAD.zip 도 새로)
