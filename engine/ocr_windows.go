//go:build windows

package main

// 스캔한 쪽 글자 인식 (Windows) — Windows 10 · 11 에 들어 있는 글자 인식(Windows.Media.Ocr)을 PowerShell 로 불러요.
// PowerShell 은 PDF 한 권에 한 번만 켜고, 그림 파일 경로를 한 줄씩 보내면 인식한 글자를 JSON 한 줄로 돌려줘요.
// 한국어 인식은 Windows 에 '한국어' 언어가 있으면 되고(한국어 Windows 는 기본으로 있어요), 없으면 영어 · 기본 언어로 읽어요.

import (
	"bufio"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const ocrScript = `$ErrorActionPreference = 'Stop'
[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType = WindowsRuntime]
$null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType = WindowsRuntime]
$null = [Windows.Graphics.Imaging.BitmapDecoder, Windows.Graphics, ContentType = WindowsRuntime]
$null = [Windows.Globalization.Language, Windows.Globalization, ContentType = WindowsRuntime]
$asTask = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1' })[0]
function Await($op, [Type]$t) { $task = $asTask.MakeGenericMethod($t).Invoke($null, @($op)); $task.Wait(-1) | Out-Null; $task.Result }
$engines = @()
foreach ($tag in @('ko', 'ko-KR')) {
  try { $l = New-Object Windows.Globalization.Language $tag; if ([Windows.Media.Ocr.OcrEngine]::IsLanguageSupported($l)) { $engines += [Windows.Media.Ocr.OcrEngine]::TryCreateFromLanguage($l); break } } catch {}
}
try { $l = New-Object Windows.Globalization.Language 'en-US'; if ([Windows.Media.Ocr.OcrEngine]::IsLanguageSupported($l)) { $engines += [Windows.Media.Ocr.OcrEngine]::TryCreateFromLanguage($l) } } catch {}
if ($engines.Count -eq 0) { $e = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages(); if ($e) { $engines += $e } }
$langs = ($engines | ForEach-Object { $_.RecognizerLanguage.LanguageTag }) -join ','
[Console]::Out.WriteLine((@{ ready = $true; langs = $langs } | ConvertTo-Json -Compress))
[Console]::Out.Flush()
while ($true) {
  $path = [Console]::In.ReadLine()
  if ($path -eq $null -or $path -eq '') { break }
  $text = ''
  try {
    $file = Await ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path)) ([Windows.Storage.StorageFile])
    $stream = Await ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
    $dec = Await ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
    $bmp = Await ($dec.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
    $best = ''
    foreach ($en in $engines) {
      $res = Await ($en.RecognizeAsync($bmp)) ([Windows.Media.Ocr.OcrResult])
      $lines = @($res.Lines | ForEach-Object { $_.Text })
      $t = $lines -join "` + "`" + `n"
      if ($t.Length -gt $best.Length * 1.3) { $best = $t }
      if ($en.RecognizerLanguage.LanguageTag -like 'ko*' -and $t.Length -gt 40) { break }
    }
    $text = $best
    $stream.Dispose()
  } catch { $text = '' }
  [Console]::Out.WriteLine((@{ text = $text } | ConvertTo-Json -Compress))
  [Console]::Out.Flush()
}
`

type ocrSession struct {
	cmd   *exec.Cmd
	in    io.WriteCloser
	out   *bufio.Reader
	dir   string
	ready bool
	n     int
}

func newOCRSession() *ocrSession {
	s := &ocrSession{}
	if env("NO_OCR") == "1" {
		return s
	}
	dir, err := os.MkdirTemp("", "sermonroad-ocr-")
	if err != nil {
		return s
	}
	s.dir = dir
	script := filepath.Join(dir, "ocr.ps1")
	// PowerShell 5.1 은 BOM 이 있어야 UTF-8 스크립트로 읽어요
	if os.WriteFile(script, append([]byte{0xEF, 0xBB, 0xBF}, []byte(ocrScript)...), 0o644) != nil {
		return s
	}
	ps := filepath.Join(os.Getenv("SystemRoot"), `System32\WindowsPowerShell\v1.0\powershell.exe`)
	if !fileExists(ps) {
		ps = "powershell.exe"
	}
	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script)
	hideWindow(cmd)
	in, err := cmd.StdinPipe()
	if err != nil {
		return s
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return s
	}
	cmd.Stderr = io.Discard
	if cmd.Start() != nil {
		return s
	}
	s.cmd, s.in, s.out = cmd, in, bufio.NewReaderSize(out, 1<<20)
	line, err := s.out.ReadString('\n')
	var hello struct {
		Ready bool   `json:"ready"`
		Langs string `json:"langs"`
	}
	if err == nil && json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(line), "\ufeff")), &hello) == nil && hello.Ready && hello.Langs != "" {
		s.ready = true
		os.Stderr.WriteString("글자 인식 언어: " + hello.Langs + "\n")
	} else {
		s.close()
	}
	return s
}

func (s *ocrSession) ok() bool { return s != nil && s.ready }

func (s *ocrSession) recognize(img image.Image) string {
	if !s.ok() {
		return ""
	}
	s.n++
	p := filepath.Join(s.dir, "page.png")
	f, err := os.Create(p)
	if err != nil {
		return ""
	}
	err = (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(f, img)
	f.Close()
	if err != nil {
		return ""
	}
	if _, err := io.WriteString(s.in, p+"\n"); err != nil {
		s.close()
		return ""
	}
	line, err := s.out.ReadString('\n')
	if err != nil {
		s.close()
		return ""
	}
	var r struct {
		Text string `json:"text"`
	}
	json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(line), "\ufeff")), &r)
	return r.Text
}

func (s *ocrSession) close() {
	if s == nil {
		return
	}
	s.ready = false
	if s.in != nil {
		io.WriteString(s.in, "\n")
		s.in.Close()
		s.in = nil
	}
	if s.cmd != nil {
		done := make(chan struct{})
		go func() { s.cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.cmd.Process.Kill()
		}
		s.cmd = nil
	}
	if s.dir != "" {
		os.RemoveAll(s.dir)
		s.dir = ""
	}
}
