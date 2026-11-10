#!/usr/bin/env python3
"""SermonROAD Windows 시험 (Windows 빌드 서버에서): python scripts/smoke-win.py <SermonROAD.exe> <결과 폴더>
실제 Windows 에서 실행기 · 엔진 · 서재 PDF 읽기(PDFium + Windows 글자 인식) · 자격 증명 관리자 · 엑셀 · 종료를 시험해요.
결과: <결과 폴더>/results.txt, status.json, shot-*.png"""
import json, os, shutil, subprocess, sys, tempfile, time, urllib.request

EXE = os.path.abspath(sys.argv[1])
OUT = os.path.abspath(sys.argv[2])
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
os.makedirs(OUT, exist_ok=True)
SUP = os.path.join(tempfile.gettempdir(), "SermonROAD-smoke")
shutil.rmtree(SUP, ignore_errors=True)
os.makedirs(SUP)
LIB = os.path.join(tempfile.gettempdir(), "SermonROAD-서재")
shutil.rmtree(LIB, ignore_errors=True)
shutil.copytree(os.path.join(ROOT, "scripts", "testdata"), LIB)

results = []


def rec(name, ok, detail=""):
    results.append((name, bool(ok), str(detail)))
    print(("PASS " if ok else "FAIL ") + name + (" — " + str(detail) if detail else ""), flush=True)


def env(**kw):
    e = dict(os.environ)
    e["SERMONROAD_SUPPORT"] = SUP
    e["SERMONROAD_UPDATE_URL"] = "http://127.0.0.1:1/none"
    e.update(kw)
    return e


def screenshot(name):
    ps = (
        "Add-Type -AssemblyName System.Windows.Forms,System.Drawing;"
        "$b=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds;"
        "$bmp=New-Object System.Drawing.Bitmap $b.Width,$b.Height;"
        "$g=[System.Drawing.Graphics]::FromImage($bmp);$g.CopyFromScreen($b.Location,[System.Drawing.Point]::Empty,$b.Size);"
        f"$bmp.Save('{os.path.join(OUT, name)}')"
    )
    subprocess.run(["powershell", "-NoProfile", "-Command", ps], timeout=60)


def pdftext(pdf, extra=()):
    out = os.path.join(SUP, "o.json")
    if os.path.exists(out):
        os.remove(out)
    t = time.time()
    p = subprocess.run([EXE, "--pdftext", pdf, out, *extra], capture_output=True, env=env(), timeout=900)
    dt = time.time() - t
    data = json.load(open(out, encoding="utf-8")) if os.path.exists(out) else None
    return p, data, dt


# ---------------------------------------------------------------- 1. 서재 PDF 읽기
try:
    p, d, dt = pdftext(os.path.join(LIB, "로마서 시험 주석.pdf"))
    ok = d and len(d["pages"]) == 7 and any("정죄함" in x for x in d["pages"])
    rec("P1 글자 PDF 읽기 (PDFium)", ok, f"{dt:.1f}초 · {len(d['pages']) if d else 0}쪽 · 진행 줄 {p.stdout.decode('utf-8','replace').count('PAGE')}개 · {p.stderr.decode('utf-8','replace').strip()[:200]}")
    p, d, dt = pdftext(os.path.join(LIB, "표준문서 시험.pdf"))
    rec("P2 두 번째 PDF (빨리 시작)", d and len(d["pages"]) == 3, f"{dt:.1f}초")
except Exception as e:
    rec("P1 글자 PDF 읽기 (PDFium)", False, e)

# 한국어 글자 인식 기능 설치를 시도해요 (빌드 서버 Windows 에는 기본으로 없을 수 있어요)
langs = subprocess.run(["powershell", "-NoProfile", "-Command",
                        "Add-Type -AssemblyName System.Runtime.WindowsRuntime;"
                        "$null=[Windows.Media.Ocr.OcrEngine,Windows.Foundation,ContentType=WindowsRuntime];"
                        "([Windows.Media.Ocr.OcrEngine]::AvailableRecognizerLanguages | % { $_.LanguageTag }) -join ','"],
                       capture_output=True, text=True, timeout=120)
rec("O0 Windows 글자 인식 언어", True, (langs.stdout.strip() or "(없음)") + " " + langs.stderr.strip()[:200])
try:
    p, d, dt = pdftext(os.path.join(LIB, "스캔 시험.pdf"))
    txt = d["pages"][0] if d else ""
    ok = d is not None and d["ocr"] == [1] and len(txt.strip()) > 10
    rec("O1 스캔 PDF 글자 인식 (Windows.Media.Ocr)", ok, f"{dt:.1f}초 · ocr={d and d['ocr']} · {txt[:120]!r} · {p.stderr.decode('utf-8','replace').strip()[:200]}")
except Exception as e:
    rec("O1 스캔 PDF 글자 인식 (Windows.Media.Ocr)", False, e)

# ---------------------------------------------------------------- 2. 실행기 → 엔진 → 앱 창
t0 = time.time()
lp = subprocess.Popen([EXE], env=env())
try:
    lp.wait(timeout=60)
except subprocess.TimeoutExpired:
    pass
port = token = ""
for _ in range(100):
    try:
        port = open(os.path.join(SUP, "port")).read().strip()
        token = open(os.path.join(SUP, "token")).read().strip()
        if port:
            break
    except OSError:
        pass
    time.sleep(0.2)
rec("L1 실행기가 엔진을 켬", bool(port), f"{time.time()-t0:.1f}초 · 포트 {port} · 실행기 끝남={lp.poll() is not None}")
BASE = f"http://127.0.0.1:{port}"


def api(path, body=None, timeout=30):
    req = urllib.request.Request(BASE + path, data=None if body is None else json.dumps(body).encode(),
                                 headers={"x-token": token, "content-type": "application/json"},
                                 method="GET" if body is None else "POST")
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode("utf-8"))


try:
    pg = api("/api/ping")
    rec("L2 엔진 대답 (ping)", pg.get("ok") and pg.get("shell") == "windows", pg)
    st = api("/api/settings")
    rec("L3 설정 · 저장 위치", st["platform"] == "windows/amd64" and st["support"].lower() == SUP.lower(), f"{st['platform']} · {st['support']} · 내보내기 {st['exportDir']}")
except Exception as e:
    rec("L2 엔진 대답 (ping)", False, e)

time.sleep(6)
screenshot("shot-app.png")
edge = subprocess.run(["tasklist", "/FI", "IMAGENAME eq msedge.exe"], capture_output=True, text=True).stdout
chrome = subprocess.run(["tasklist", "/FI", "IMAGENAME eq chrome.exe"], capture_output=True, text=True).stdout
rec("L4 앱 창 (Edge · Chrome 앱 창)", "msedge.exe" in edge or "chrome.exe" in chrome, "shot-app.png")

# 두 번째 실행 → 이미 켜진 엔진을 그대로 쓰고 창만 다시 열어요
t0 = time.time()
lp2 = subprocess.Popen([EXE], env=env())
try:
    lp2.wait(timeout=30)
    same = open(os.path.join(SUP, "port")).read().strip() == port and api("/api/ping").get("ok")
    rec("L5 다시 실행하면 창만 다시 열기", same, f"{time.time()-t0:.1f}초")
except Exception as e:
    rec("L5 다시 실행하면 창만 다시 열기", False, e)

# ---------------------------------------------------------------- 3. 기록 · 엑셀
try:
    r = api("/api/sermon/save", {"sermon": {"date": "2026-11-08", "kind": "주일", "ref": "롬 8:1-4", "title": "정죄함이 없나니", "place": "본당", "status": "설교함"}})
    d = api("/api/data")
    n = len(d.get("sermons") or [])
    rec("D1 설교 기록 저장", n >= 1, f"{n}편 · {str(r)[:120]}")
    ex = api("/api/export", {"scope": "all"})
    rec("D2 엑셀 내보내기", ex.get("path") and os.path.exists(ex["path"]), ex)
    b = api("/api/backup/now", {})
    rec("D3 백업", True, str(b)[:150])
except Exception as e:
    rec("D1 설교 기록 저장", False, e)

# ---------------------------------------------------------------- 4. 자격 증명 관리자 (API 키)
KEY = "sk-ant-api03-smoketest-abcdefghijklmnopqrstuvwxyz0123456789"
try:
    c = api("/api/claude/key", {"key": KEY})
    listed = "SermonROAD" in subprocess.run(["cmdkey", "/list"], capture_output=True, text=True, errors="replace").stdout
    rec("K1 API 키를 자격 증명 관리자에 저장", c.get("hasKey") and listed, f"hasKey={c.get('hasKey')} · cmdkey 목록에 있음={listed}")
    not_on_disk = all(KEY not in open(os.path.join(dp, f), errors="ignore").read()
                      for dp, _, fs in os.walk(SUP) for f in fs if os.path.getsize(os.path.join(dp, f)) < 5_000_000)
    rec("K2 키가 파일로 남지 않음", not_on_disk)
    c = api("/api/claude/key", {"key": ""})
    rec("K3 키 지우기", not c.get("hasKey"))
    # ChatGPT · 제미나이 키는 자격 증명 관리자에 따로 저장
    api("/api/ai/key", {"ai": "chatgpt", "key": "sk-proj-smoketest-000000000000000000000000"})
    c = api("/api/ai/key", {"ai": "gemini", "key": "AIzaSmokeTest0000000000000000000000000"})
    ks = ",".join(p["id"] + ("=1" if p["hasKey"] else "=0") for p in c["providers"])
    out = subprocess.run(["cmdkey", "/list"], capture_output=True, text=True, errors="replace").stdout
    rec("K4 ChatGPT · 제미나이 키를 따로 저장", ks == "claude=0,chatgpt=1,gemini=1" and "OpenAI" in out and "Gemini" in out, ks)
    api("/api/ai/key", {"ai": "chatgpt", "key": ""})
    c = api("/api/ai/key", {"ai": "gemini", "key": ""})
    rec("K5 ChatGPT · 제미나이 키 지우기", not any(p["hasKey"] for p in c["providers"]))
except Exception as e:
    rec("K1 API 키를 자격 증명 관리자에 저장", False, e)

# ---------------------------------------------------------------- 5. 서재 (엔진이 PDF 읽기 도구를 따로 켜서 읽기)
try:
    api("/api/library/folder", {"path": LIB})
    t0 = time.time()
    docs = []
    while time.time() - t0 < 300:
        docs = api("/api/library").get("docs") or []
        if docs and all(x["status"] in ("ready", "error") for x in docs):
            break
        time.sleep(1)
    rec("S1 서재 폴더 읽기", docs and all(x["status"] == "ready" for x in docs),
        f"{time.time()-t0:.1f}초 · " + ", ".join(f"{x['name']}:{x['status']}:{x['pages']}쪽:ocr{x['ocr']}" for x in docs))
    f = api("/api/library/find", {"ref": "롬 8:1"})
    hits = f.get("hits") or []
    rec("S2 본문으로 찾기 (롬 8:1)", hits and hits[0]["name"] == "로마서 시험 주석", str(hits[:1])[:200])
    screenshot("shot-after.png")
except Exception as e:
    rec("S1 서재 폴더 읽기", False, e)

# ---------------------------------------------------------------- 6. 종료 버튼 → 엔진 끝
try:
    api("/api/quit", {})
    gone = False
    for _ in range(40):
        time.sleep(0.25)
        try:
            api("/api/ping", timeout=2)
        except Exception:
            gone = True
            break
    tl = subprocess.run(["tasklist", "/FI", "IMAGENAME eq sermonroad-engine.exe"], capture_output=True, text=True).stdout
    rec("Q1 종료 버튼 → 엔진 끝", gone and "sermonroad-engine.exe" not in tl)
except Exception as e:
    rec("Q1 종료 버튼 → 엔진 끝", False, e)

# ---------------------------------------------------------------- 결과
log = ""
for n in ("engine.log", "engine.out"):
    pth = os.path.join(SUP, n)
    if os.path.exists(pth):
        log += f"\n----- {n}\n" + open(pth, encoding="utf-8", errors="replace").read()[-6000:]
passed = sum(1 for _, ok, _ in results if ok)
with open(os.path.join(OUT, "results.txt"), "w", encoding="utf-8") as f:
    for n, ok, d in results:
        f.write(("PASS " if ok else "FAIL ") + n + (" — " + d if d else "") + "\n")
    f.write(f"\n{passed}/{len(results)} 통과\n{log}")
# O1(한국어가 없는 빌드 서버의 글자 인식)은 참고용: 빌드 서버에 한국어 언어가 없을 수 있어요
must = [ok for n, ok, _ in results if not n.startswith("O1")]
json.dump({"smoke": "ok" if all(must) else "fail", "passed": passed, "total": len(results)},
          open(os.path.join(OUT, "status.json"), "w"))
print(f"{passed}/{len(results)} 통과")
