#!/usr/bin/env python3
"""zipdir.py <폴더> <결과.zip> [안에 넣을 첫 폴더 이름] — 폴더 안 파일을 UTF-8 이름으로 압축해요 (Windows 탐색기에서 한글 이름이 바르게 보여요)"""
import os, sys, zipfile
src, out = sys.argv[1], sys.argv[2]
only = sys.argv[3] if len(sys.argv) > 3 else None
tmp = out + ".part"
with zipfile.ZipFile(tmp, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as z:
    for root, dirs, files in os.walk(src):
        dirs.sort()
        for f in sorted(files):
            p = os.path.join(root, f)
            rel = os.path.relpath(p, src).replace(os.sep, "/")
            if only and not (rel == only or rel.startswith(only + "/")):
                continue
            z.write(p, rel)
os.replace(tmp, out)
print(f"✔ {out}")
