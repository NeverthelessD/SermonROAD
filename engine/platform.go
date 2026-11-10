package main

import "runtime"

// 맥(Apple Silicon) 판과 Windows 판(engine/launcher_windows.go · pdftext_pdfium.go · ocr_windows.go)이 같은 엔진을 써요.
var isWindows = runtime.GOOS == "windows"
