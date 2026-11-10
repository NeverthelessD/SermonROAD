//go:build !darwin

package main

// 서재 PDF 글자 읽기 (Windows 판) — 맥의 sermonroad-pdftext 와 같은 일을 해요.
//   SermonROAD.exe --pdftext <PDF> <결과.json>
//   PDFium(크롬이 쓰는 PDF 엔진, WebAssembly 로 앱 안에 들어 있어요)으로 쪽마다 글자를 읽고,
//   글자가 거의 없는 쪽(스캔한 책)은 그림으로 그려서 Windows 에 들어 있는 글자 인식(Windows.Media.Ocr)으로 읽어요.
//   진행 상황은 표준 출력에 "PAGE 3/120" 처럼 한 줄씩 알려요 (엔진이 읽어서 화면에 보여 줘요).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
)

// pdfTextMain runs the --pdftext tool and returns the process exit code.
func pdfTextMain(args []string) int {
	allowOCR := true
	var files []string
	for _, a := range args {
		if a == "--no-ocr" {
			allowOCR = false
		} else {
			files = append(files, a)
		}
	}
	if len(files) != 2 {
		fmt.Fprintln(os.Stderr, "사용법: --pdftext <PDF> <결과.json> [--no-ocr]")
		return 1
	}
	return runPDFText(files[0], files[1], allowOCR)
}

func meaningfulLetters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}

func runPDFText(src, out string, allowOCR bool) int {
	fail := func(code int, msg string) int {
		fmt.Fprintln(os.Stderr, msg)
		return code
	}
	f, err := os.Open(src)
	if err != nil {
		return fail(2, "PDF를 열지 못했어요")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return fail(2, "PDF를 열지 못했어요")
	}

	cfg := webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 1}
	// 한 번 준비한 PDF 엔진을 저장해 두면 다음 PDF 부터는 빨리 시작해요
	if sup := env("SUPPORT"); sup != "" {
		if cache, err := wazero.NewCompilationCacheWithDir(filepath.Join(sup, "pdfium-cache")); err == nil {
			cfg.RuntimeConfig = wazero.NewRuntimeConfig().WithCompilationCache(cache)
		}
	}
	pool, err := webassembly.Init(cfg)
	if err != nil {
		return fail(2, "PDF 읽기 도구를 준비하지 못했어요: "+err.Error())
	}
	defer pool.Close()
	inst, err := pool.GetInstance(time.Minute)
	if err != nil {
		return fail(2, "PDF 읽기 도구를 준비하지 못했어요: "+err.Error())
	}
	defer inst.Close()

	doc, err := inst.OpenDocument(&requests.OpenDocument{FileReader: f, FileReaderSize: st.Size()})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "password") {
			return fail(3, "암호가 걸린 PDF예요")
		}
		return fail(2, "PDF를 열지 못했어요")
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	pc, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return fail(2, "PDF를 열지 못했어요")
	}
	n := pc.PageCount

	title := ""
	if md, err := inst.GetMetaData(&requests.GetMetaData{Document: doc.Document, Tags: &[]string{"Title"}}); err == nil {
		for _, t := range md.Tags {
			if t.Tag == "Title" {
				title = strings.TrimSpace(t.Value)
			}
		}
	}

	var ocr *ocrSession
	pages := make([]string, n)
	ocrPages := []int{}
	for i := 0; i < n; i++ {
		page := requests.Page{ByIndex: &requests.PageByIndex{Document: doc.Document, Index: i}}
		text := ""
		if r, err := inst.GetPageText(&requests.GetPageText{Page: page}); err == nil {
			text = r.Text
		}
		if allowOCR && meaningfulLetters(text) < 12 {
			if ocr == nil {
				ocr = newOCRSession()
			}
			if ocr.ok() {
				if img, err := inst.RenderPageInDPI(&requests.RenderPageInDPI{Page: page, DPI: 200}); err == nil && img.Result.Image != nil {
					if t := ocr.recognize(img.Result.Image); strings.TrimSpace(t) != "" {
						text = t
						ocrPages = append(ocrPages, i+1)
					}
					img.Cleanup()
				}
			}
		}
		pages[i] = text
		fmt.Printf("PAGE %d/%d\n", i+1, n)
		os.Stdout.Sync()
	}
	if ocr != nil {
		ocr.close()
	}
	b, err := json.Marshal(map[string]any{"pages": pages, "ocr": ocrPages, "title": title})
	if err != nil {
		return 4
	}
	tmp := out + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return 5
	}
	if err := os.Rename(tmp, out); err != nil {
		return 5
	}
	return 0
}

var _ pdfium.Pool // keep the import explicit for readers
