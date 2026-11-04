#!/usr/bin/env python3
"""시험용 서재 PDF 만들기 (Playwright + Chromium) → scripts/testdata/
   - 로마서 시험 주석.pdf : 글자가 있는 PDF (장 제목 · 절 표기)
   - 표준문서 시험.pdf    : 다른 책 · 본문 표기
   - 스캔 시험.pdf        : 글자 없이 그림만 있는 PDF (맥에서 글자 인식 시험)
   내용은 시험을 위해 새로 쓴 글이에요."""
import asyncio, base64, os
from playwright.async_api import async_playwright

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "scripts", "testdata")
FONT = os.path.join(ROOT, "engine", "web", "fonts", "PretendardVariable.woff2")

CSS = """@font-face{font-family:P;src:url(data:font/woff2;base64,%s) format('woff2')}
body{font-family:P,sans-serif;font-size:15px;line-height:1.7;margin:0}
.pg{page-break-after:always;padding:30px 40px}h1{font-size:22px}h2{font-size:18px}"""

ROMANS = [
    ("제7장", "율법과 죄에 대하여. 7:14 이하에서 바울은 자신 안의 싸움을 말한다. 이 장은 8장으로 넘어가는 다리가 된다."),
    ("제8장", "8:1 그러므로 이제 그리스도 예수 안에 있는 자에게는 결코 정죄함이 없다. 1절은 7장의 탄식에 대한 하나님의 대답이다."),
    ("", "18절부터 27절까지는 현재의 고난과 장차 나타날 영광을 견준다. 피조물도 함께 탄식하며, 성령께서도 우리를 위하여 간구하신다."),
    ("", "28절. '모든 것이 합력하여 선을 이룬다'는 말씀은 막연한 낙관이 아니다. 선을 이루시는 분은 하나님이시다. 그 대상은 하나님을 사랑하는 자, 곧 그의 뜻대로 부르심을 입은 자들이다."),
    ("", "29-30절은 구원의 다섯 고리를 말한다. 미리 아심, 미리 정하심, 부르심, 의롭다 하심, 영화롭게 하심. 다섯 동사가 모두 과거형으로 쓰여 확실성을 드러낸다. 롬 8:28-30 을 엡 1:4-5 와 함께 읽으라."),
    ("", "31절 이하는 승리의 찬송이다. 누가 우리를 그리스도의 사랑에서 끊으리요. 8:35-39"),
    ("제9장", "9:1-5 바울은 동족 이스라엘을 위하여 큰 근심을 품는다."),
]
STANDARDS = [
    ("제3장 하나님의 영원한 작정", "하나님은 영원 전부터 자기 뜻의 지극히 지혜롭고 거룩한 계획에 따라 일어날 모든 일을 정하셨다. 근거 구절: 엡 1:11, 롬 11:33, 롬 8:28-30."),
    ("제10장 효력 있는 부르심", "하나님은 생명에 이르도록 예정하신 모든 자를 그의 말씀과 성령으로 부르신다. 근거 구절: 롬 8:30, 롬 11:7, 엡 1:10-11."),
    ("제17장 성도의 견인", "하나님께서 받아들이신 자들은 은혜의 상태에서 완전히 떨어져 나갈 수 없다. 근거 구절: 빌 1:6, 요 10:28-29, 롬 8:35-39."),
]


def pages_html(pages):
    return "".join(f"<div class='pg'>{'<h1>' + h + '</h1>' if h else ''}<p>{t}</p></div>" for h, t in pages)


async def main():
    os.makedirs(OUT, exist_ok=True)
    css = CSS % base64.b64encode(open(FONT, "rb").read()).decode()
    async with async_playwright() as p:
        b = await p.chromium.launch()
        pg = await b.new_page()
        for name, pages in [("로마서 시험 주석.pdf", ROMANS), ("표준문서 시험.pdf", STANDARDS)]:
            await pg.set_content(f"<html><head><meta charset=utf-8><style>{css}</style></head><body>{pages_html(pages)}</body></html>")
            await pg.wait_for_timeout(300)
            await pg.pdf(path=os.path.join(OUT, name), format="A5", print_background=True)
        # 스캔 PDF: 글자를 그림으로 찍어서 그림만 넣어요
        await pg.set_viewport_size({"width": 900, "height": 600})
        await pg.set_content(f"<html><head><meta charset=utf-8><style>{css} body{{background:#fff;padding:40px;font-size:30px}}</style></head><body><h1>로마서 강해 제8장</h1><p>롬 8:28 모든 것이 합력하여 선을 이루느니라</p><p>하나님을 사랑하는 자들에게</p></body></html>")
        await pg.wait_for_timeout(300)
        png = base64.b64encode(await pg.screenshot()).decode()
        await pg.set_content(f"<html><body style='margin:0'><img src='data:image/png;base64,{png}' style='width:100%'></body></html>")
        await pg.pdf(path=os.path.join(OUT, "스캔 시험.pdf"), format="A5", landscape=True)
        await b.close()
    print("✔", ", ".join(sorted(os.listdir(OUT))))

asyncio.run(main())
