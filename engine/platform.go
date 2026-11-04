package main

import "runtime"

// SermonROAD 는 맥(Apple Silicon) 전용이에요. 업데이트 코드가 Windows 경우도 품고 있어서 남겨 둬요.
var isWindows = runtime.GOOS == "windows"
