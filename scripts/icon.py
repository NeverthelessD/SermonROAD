#!/usr/bin/env python3
"""app/AppIcon.svg → app/AppIcon.png (1024) · app/AppIcon.icns · engine/web/icon.png (화면 로고용 256)
(Playwright + Chromium 필요) · python3 scripts/icon.py"""
import asyncio, base64, io, os, struct
from playwright.async_api import async_playwright

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SVG = os.path.join(ROOT, "app", "AppIcon.svg")
# icns 의 PNG 칸: (종류, 크기)
TYPES = [("icp4", 16), ("icp5", 32), ("icp6", 64), ("ic07", 128), ("ic08", 256), ("ic09", 512), ("ic10", 1024),
         ("ic11", 32), ("ic12", 64), ("ic13", 256), ("ic14", 512)]


async def render(page, size):
    svg = base64.b64encode(open(SVG, "rb").read()).decode()
    await page.set_viewport_size({"width": size, "height": size})
    await page.set_content(f"<html><body style='margin:0;background:transparent'>"
                           f"<img src='data:image/svg+xml;base64,{svg}' width='{size}' height='{size}' style='display:block'></body></html>")
    await page.wait_for_timeout(150)
    return await page.screenshot(omit_background=True, clip={"x": 0, "y": 0, "width": size, "height": size})


async def main():
    async with async_playwright() as p:
        b = await p.chromium.launch()
        page = await b.new_page(device_scale_factor=1)
        pngs = {}
        for s in sorted({s for _, s in TYPES}):
            pngs[s] = await render(page, s)
        await b.close()
    open(os.path.join(ROOT, "app", "AppIcon.png"), "wb").write(pngs[1024])
    open(os.path.join(ROOT, "engine", "web", "icon.png"), "wb").write(pngs[256])
    body = b"".join(t.encode() + struct.pack(">I", 8 + len(pngs[s])) + pngs[s] for t, s in TYPES)
    open(os.path.join(ROOT, "app", "AppIcon.icns"), "wb").write(b"icns" + struct.pack(">I", 8 + len(body)) + body)
    print("✔ AppIcon.png · AppIcon.icns · web/icon.png")

asyncio.run(main())
