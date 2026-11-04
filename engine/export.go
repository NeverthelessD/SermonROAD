package main

// 내보내기: 엑셀(.xlsx) · 인쇄용 목록(브라우저에서 ⌘P)

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var dowKo = []string{"일", "월", "화", "수", "목", "금", "토"}

func confLabel(c ConfLink) string {
	switch c.Doc {
	case "hc":
		return fmt.Sprintf("하이델베르크 제%d주일", c.N)
	case "wsc":
		return fmt.Sprintf("소요리문답 %d문", c.N)
	case "wcf":
		return fmt.Sprintf("신앙고백 %d장", c.N)
	case "dort":
		names := []string{"", "도르트 제1교리", "도르트 제2교리", "도르트 제3·4교리", "도르트 제5교리"}
		if c.N >= 1 && c.N <= 4 {
			return names[c.N]
		}
	}
	return fmt.Sprintf("%s %d", c.Doc, c.N)
}

// selectSermons: scope = all | year | month — 날짜 순
func (s *Store) selectSermons(scope string, y, m int) []*Sermon {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Sermon
	for _, x := range s.data.Sermons {
		ok := true
		switch scope {
		case "year":
			ok = strings.HasPrefix(x.Date, fmt.Sprintf("%04d-", y))
		case "month":
			ok = strings.HasPrefix(x.Date, fmt.Sprintf("%04d-%02d-", y, m))
		}
		if ok {
			c := *x
			out = append(out, &c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].Time < out[j].Time
	})
	return out
}

func (s *Store) seriesName(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.data.Series {
		if x.ID == id {
			return x.Name
		}
	}
	return ""
}

func scopeLabel(scope string, y, m int) string {
	switch scope {
	case "year":
		return fmt.Sprintf("%d년", y)
	case "month":
		return fmt.Sprintf("%d년 %d월", y, m)
	}
	return "전체"
}

func dowOf(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return dowKo[t.Weekday()]
}

func (a *App) exportRows(list []*Sermon) [][]string {
	rows := [][]string{{"날짜", "요일", "시간", "종류", "절기", "본문", "제목", "장소", "강해 시리즈", "상태", "중심 명제", "대지", "요약 · 메모", "요리문답 · 신앙고백", "참고한 자료"}}
	for _, x := range list {
		kind := x.Kind
		if kind == "그 외" && x.KindOther != "" {
			kind = x.KindOther
		}
		ser := a.store.seriesName(x.SeriesID)
		if ser != "" && x.SeriesNo > 0 {
			ser += " " + strconv.Itoa(x.SeriesNo)
		}
		var conf []string
		for _, c := range x.Confessions {
			conf = append(conf, confLabel(c))
		}
		ref := refsText(x.Passages)
		if ref == "" {
			ref = x.PassageText
		}
		rows = append(rows, []string{x.Date, dowOf(x.Date), x.Time, kind, x.Season, ref, x.Title, x.Place, ser, x.Status, x.Thesis,
			strings.Join(x.Points, "\n"), x.Summary, strings.Join(conf, ", "), x.Refs})
	}
	return rows
}

func xmlEsc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0xFFFE || r == 0xFFFF:
			// 엑셀이 받지 않는 글자는 빼요
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// writeXLSX: 글자만 담은 간단한 엑셀 파일 (첫 줄은 굵게, 틀 고정)
func writeXLSX(rows [][]string, sheetName string, widths []int) ([]byte, error) {
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	add := func(name, body string) error {
		w, err := z.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(body))
		return err
	}
	add("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`)
	add("_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`)
	add("xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="`+xmlEsc(sheetName)+`" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	add("xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`)
	add("xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="2"><font><sz val="11"/><name val="Apple SD Gothic Neo"/></font><font><b/><sz val="11"/><name val="Apple SD Gothic Neo"/></font></fonts><fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FFF1EADB"/><bgColor indexed="64"/></patternFill></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="3"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment vertical="top" wrapText="1"/></xf><xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"/><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs></styleSheet>`)
	var sh strings.Builder
	sh.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><cols>`)
	for i, wdt := range widths {
		fmt.Fprintf(&sh, `<col min="%d" max="%d" width="%d" customWidth="1"/>`, i+1, i+1, wdt)
	}
	sh.WriteString(`</cols><sheetData>`)
	for ri, row := range rows {
		fmt.Fprintf(&sh, `<row r="%d">`, ri+1)
		for ci, v := range row {
			style := 0
			if ri == 0 {
				style = 1
			}
			fmt.Fprintf(&sh, `<c r="%s%d" t="inlineStr" s="%d"><is><t xml:space="preserve">%s</t></is></c>`, colName(ci), ri+1, style, xmlEsc(v))
		}
		sh.WriteString(`</row>`)
	}
	sh.WriteString(`</sheetData>`)
	if len(rows) > 0 {
		fmt.Fprintf(&sh, `<autoFilter ref="A1:%s%d"/>`, colName(len(rows[0])-1), len(rows))
	}
	sh.WriteString(`</worksheet>`)
	if err := add("xl/worksheets/sheet1.xml", sh.String()); err != nil {
		return nil, err
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func exportDir() string {
	if d := env("EXPORT_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	for _, d := range []string{filepath.Join(home, "Downloads"), filepath.Join(home, "Desktop")} {
		if dirExists(d) {
			return d
		}
	}
	return home
}

func (a *App) handleExport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Scope string `json:"scope"`
		Y     int    `json:"y"`
		M     int    `json:"m"`
	}
	readJSON(r, &in)
	list := a.store.selectSermons(in.Scope, in.Y, in.M)
	if len(list) == 0 {
		jsonError(w, 400, "내보낼 설교가 없어요")
		return
	}
	label := scopeLabel(in.Scope, in.Y, in.M)
	b, err := writeXLSX(a.exportRows(list), "설교 기록", []int{12, 5, 7, 9, 12, 20, 34, 16, 18, 9, 36, 40, 50, 26, 30})
	if err != nil {
		jsonError(w, 500, "엑셀 파일을 만들지 못했어요")
		return
	}
	name := fmt.Sprintf("SermonROAD 설교 기록 %s.xlsx", label)
	p, err := uniquePath(exportDir(), name)
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		jsonError(w, 500, "파일을 저장하지 못했어요: "+err.Error())
		return
	}
	revealPaths([]string{p})
	writeJSON(w, map[string]any{"ok": true, "path": p, "count": len(list)})
}

func uniquePath(dir, name string) (string, error) {
	if !dirExists(dir) {
		return "", errors.New("저장할 폴더를 찾지 못했어요")
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	p := filepath.Join(dir, name)
	for i := 2; fileExists(p) && i < 200; i++ {
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
	}
	return p, nil
}

// /print?t=…&scope=month&y=2026&m=10 — 인쇄용 목록 (브라우저에서 열려요)
func (a *App) handlePrint(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("t") != a.token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	y, _ := strconv.Atoi(q.Get("y"))
	m, _ := strconv.Atoi(q.Get("m"))
	scope := q.Get("scope")
	list := a.store.selectSermons(scope, y, m)
	label := scopeLabel(scope, y, m)
	var b strings.Builder
	e := html.EscapeString
	fmt.Fprintf(&b, `<!doctype html><html lang="ko"><head><meta charset="utf-8"><title>설교 기록 %s</title><style>
@page{size:A4;margin:14mm}
body{font:13px/1.55 -apple-system,"Apple SD Gothic Neo","Malgun Gothic",sans-serif;color:#1B1D26;margin:24px}
h1{font-size:20px;margin:0 0 4px}.sub{color:#555A6B;margin:0 0 16px}
table{border-collapse:collapse;width:100%%}th,td{border-bottom:1px solid #E3DCCC;padding:7px 6px;text-align:left;vertical-align:top}
th{background:#F1EADB;font-size:12px}td.d{white-space:nowrap;font-variant-numeric:tabular-nums}.k{font-size:11px;color:#555A6B}
.t{font-weight:700}.btn{position:fixed;top:16px;right:16px;padding:8px 14px;border-radius:9px;border:0;background:#1E2747;color:#F4EBD3;font-weight:700;cursor:pointer}
@media print{.btn{display:none}body{margin:0}}
.foot{margin-top:18px;color:#8B91A8;font-size:11px;text-align:right}
</style></head><body><button class="btn" onclick="print()">인쇄 (⌘P)</button><h1>설교 기록 — %s</h1><p class="sub">모두 %d편</p>
<table><thead><tr><th>날짜</th><th>종류</th><th>본문</th><th>제목</th><th>장소</th><th>시리즈</th></tr></thead><tbody>`, e(label), e(label), len(list))
	for _, x := range list {
		kind := x.Kind
		if kind == "그 외" && x.KindOther != "" {
			kind = x.KindOther
		}
		ser := a.store.seriesName(x.SeriesID)
		if ser != "" && x.SeriesNo > 0 {
			ser += " " + strconv.Itoa(x.SeriesNo)
		}
		ref := refsText(x.Passages)
		if ref == "" {
			ref = x.PassageText
		}
		date := x.Date
		if t, err := time.Parse("2006-01-02", x.Date); err == nil {
			date = fmt.Sprintf("%d.%02d.%02d (%s)", t.Year(), t.Month(), t.Day(), dowKo[t.Weekday()])
		}
		season := ""
		if x.Season != "" {
			season = "<div class=k>" + e(x.Season) + "</div>"
		}
		fmt.Fprintf(&b, `<tr><td class=d>%s</td><td>%s%s</td><td>%s</td><td class=t>%s</td><td>%s</td><td>%s</td></tr>`,
			e(date), e(kind), season, e(ref), e(x.Title), e(x.Place), e(ser))
	}
	b.WriteString(`</tbody></table><p class="foot">SermonROAD · made by. Nevertheless_D</p></body></html>`)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(b.String()))
}
