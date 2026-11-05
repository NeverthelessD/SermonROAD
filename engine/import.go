package main

// 설교 한꺼번에 등록
//   앱을 쓰기 전에 했던 설교들을 한 번에 넣어요. 세 가지 길:
//     1) 표 붙여 넣기  : 엑셀 · 한글 · 메모장에서 복사한 줄들 (칸은 탭, | 또는 " / " 로 나뉨, 아니면 한 줄을 통째로 읽기)
//     2) 엑셀 파일     : .xlsx (첫 줄이 '날짜 · 종류 · 본문 · 제목 · 장소 …' 머리글이면 그대로, 아니면 칸 모양으로 짐작)
//     3) 원고 파일 폴더 : 파일 이름(예: 20190310 주일 롬8장28-30 하나님의 영원한 작정.hwp)에서 날짜 · 종류 · 본문 · 제목을 읽고
//                        원고 파일을 설교 기록에 첨부해요
//   먼저 '미리 보기'로 돌려주고, 사용자가 고친 뒤 '등록'하면 저장해요. 등록 전에 백업을 남기고, 바로 되돌릴 수 있어요.

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type ImportRow struct {
	Date        string `json:"date"`
	Time        string `json:"time"`
	Kind        string `json:"kind"`
	KindOther   string `json:"kindOther"`
	PassageText string `json:"passageText"`
	Ref         string `json:"ref"`   // 알아본 본문 (못 알아보면 빈칸)
	RefOK       bool   `json:"refOK"` // 본문을 알아봤는지
	Title       string `json:"title"`
	Place       string `json:"place"`
	Summary     string `json:"summary"`
	Season      string `json:"season"`
	Series      string `json:"series"` // 시리즈 이름
	Status      string `json:"status"`
	File        string `json:"file"`     // 첨부할 원고 파일 (폴더 방식)
	FileName    string `json:"fileName"` // 보여 줄 파일 이름
	Src         string `json:"src"`      // 읽은 원래 글 (확인용)
	Dup         string `json:"dup"`      // 이미 있는 기록과 겹치면 그 설명
	Warn        string `json:"warn"`
}

// ---------------------------------------------------------------- 날짜 · 종류 · 본문 읽기 (느슨하게)

var (
	reDateFull  = regexp.MustCompile(`((?:19|20)\d{2})\s?[.\-/년]\s?(\d{1,2})\s?[.\-/월]\s?(\d{1,2})\s?일?`)
	reDate8     = regexp.MustCompile(`(?:^|\D)((?:19|20)\d{2})(\d{2})(\d{2})(?:\D|$)`)
	reDate6     = regexp.MustCompile(`^\s*(\d{2})[.\-]?(\d{2})[.\-]?(\d{2})(?:\D|$)`)
	reDateShort = regexp.MustCompile(`(?:^|\D)(\d{2})\s?[.\-/]\s?(\d{1,2})\s?[.\-/]\s?(\d{1,2})(?:\D|$)`)
	reMonthDay  = regexp.MustCompile(`(?:^|\D)(\d{1,2})\s?(?:월\s?(\d{1,2})\s?일|[.\-/](\d{1,2}))(?:\D|$)`)
	reMMDD      = regexp.MustCompile(`^\s*(\d{2})(\d{2})(?:\D|$)`)
	reTimeCell  = regexp.MustCompile(`^\s*(\d{1,2}):(\d{2})\s*$`)
	reNoise     = regexp.MustCompile(`(?i)\(\d+\)|최종본?|수정본?|원고|설교문|_?final|copy|사본`)
	reTrimSep   = regexp.MustCompile(`^[\s\-_.,·:;|/()\[\]{}「」『』<>]+|[\s\-_.,·:;|/()\[\]{}「」『』<>]+$`)
	reSpace     = regexp.MustCompile(`\s+`)
)

func validDate(y, m, d int) (string, bool) {
	if y < 1950 || y > 2100 || m < 1 || m > 12 || d < 1 || d > 31 {
		return "", false
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.Local)
	if t.Month() != time.Month(m) {
		return "", false
	}
	return t.Format("2006-01-02"), true
}

// takeDate: 글에서 날짜를 찾아 떼어 내요 (yearHint: 폴더 이름 등에서 얻은 해)
func takeDate(s string, yearHint int) (string, string) {
	try := func(re *regexp.Regexp, f func(m []string) (int, int, int)) (string, string, bool) {
		loc := re.FindStringSubmatchIndex(s)
		if loc == nil {
			return "", s, false
		}
		m := make([]string, len(loc)/2)
		for i := range m {
			if loc[2*i] >= 0 {
				m[i] = s[loc[2*i]:loc[2*i+1]]
			}
		}
		y, mo, d := f(m)
		if ds, ok := validDate(y, mo, d); ok {
			return ds, s[:loc[0]] + " " + s[loc[1]:], true
		}
		return "", s, false
	}
	if d, r, ok := try(reDateFull, func(m []string) (int, int, int) { return atoi(m[1]), atoi(m[2]), atoi(m[3]) }); ok {
		return d, r
	}
	if d, r, ok := try(reDate8, func(m []string) (int, int, int) { return atoi(m[1]), atoi(m[2]), atoi(m[3]) }); ok {
		return d, r
	}
	if d, r, ok := try(reDate6, func(m []string) (int, int, int) { return 2000 + atoi(m[1]), atoi(m[2]), atoi(m[3]) }); ok {
		return d, r
	}
	if d, r, ok := try(reDateShort, func(m []string) (int, int, int) { return 2000 + atoi(m[1]), atoi(m[2]), atoi(m[3]) }); ok {
		return d, r
	}
	if yearHint > 0 {
		if d, r, ok := try(reMonthDay, func(m []string) (int, int, int) { return yearHint, atoi(m[1]), atoi(m[2]) + atoi(m[3]) }); ok {
			return d, r
		}
		if d, r, ok := try(reMMDD, func(m []string) (int, int, int) { return yearHint, atoi(m[1]), atoi(m[2]) }); ok {
			return d, r
		}
	}
	return "", s
}

// 종류 낱말: 앞의 것이 먼저 (주일 오후가 주일보다 먼저)
var kindWords = []struct {
	re         *regexp.Regexp
	kind, name string
}{
	{regexp.MustCompile(`주일\s?오후(?:\s?예배)?|오후\s?예배|주일\s?저녁(?:\s?예배)?|저녁\s?예배`), "주일 오후", ""},
	{regexp.MustCompile(`새벽(?:\s?기도회|\s?예배)?`), "새벽", ""},
	{regexp.MustCompile(`수요(?:\s?기도회|\s?예배|\s?저녁)?|수요일`), "수요", ""},
	{regexp.MustCompile(`금요(?:\s?기도회|\s?철야|\s?예배)?|금요일|금철`), "금요", ""},
	{regexp.MustCompile(`성찬(?:\s?예배|식)?`), "성찬", ""},
	{regexp.MustCompile(`주일\s?(?:오전|낮)?(?:\s?예배)?|오전\s?예배|대예배`), "주일", ""},
	{regexp.MustCompile(`장례(?:\s?예배)?|입관(?:\s?예배)?|발인(?:\s?예배)?|하관(?:\s?예배)?`), "그 외", "장례 예배"},
	{regexp.MustCompile(`결혼(?:\s?예배|식)?|혼인(?:\s?예배)?`), "그 외", "결혼 예배"},
	{regexp.MustCompile(`부흥회|사경회`), "그 외", "부흥회"},
	{regexp.MustCompile(`수련회`), "그 외", "수련회"},
	{regexp.MustCompile(`임직(?:\s?예배|식)?`), "그 외", "임직 예배"},
	{regexp.MustCompile(`헌신\s?예배`), "그 외", "헌신 예배"},
	{regexp.MustCompile(`송구영신(?:\s?예배)?`), "그 외", "송구영신 예배"},
	{regexp.MustCompile(`신년(?:\s?예배)?`), "그 외", "신년 예배"},
	{regexp.MustCompile(`심방(?:\s?예배)?`), "그 외", "심방 예배"},
}

var seasonWords = []struct {
	re   *regexp.Regexp
	name string
}{
	{regexp.MustCompile(`부활절|부활\s?주일`), "부활절"},
	{regexp.MustCompile(`성탄절|성탄\s?(?:예배|주일)|성탄`), "성탄절"},
	{regexp.MustCompile(`추수\s?감사(?:절|주일)?`), "추수감사절"},
	{regexp.MustCompile(`맥추\s?감사(?:절|주일)?`), "맥추감사절"},
	{regexp.MustCompile(`종교\s?개혁(?:\s?기념)?(?:주일)?`), "종교개혁주일"},
	{regexp.MustCompile(`대림절|대강절`), "대림절"},
	{regexp.MustCompile(`사순절`), "사순절"},
	{regexp.MustCompile(`고난\s?주간|종려\s?주일|성금요일`), "고난주간"},
	{regexp.MustCompile(`성령\s?강림(?:절|주일)?|오순절`), "성령강림절"},
	{regexp.MustCompile(`주현절`), "주현절"},
	{regexp.MustCompile(`승천(?:일|주일)`), "승천일"},
}

func takeSeason(s string) (string, string) {
	for _, w := range seasonWords {
		if loc := w.re.FindStringIndex(s); loc != nil {
			return w.name, s[:loc[0]] + " " + s[loc[1]:]
		}
	}
	return "", s
}

func takeKind(s string) (kind, other, rest string) {
	for _, k := range kindWords {
		if loc := k.re.FindStringIndex(s); loc != nil {
			return k.kind, k.name, s[:loc[0]] + " " + s[loc[1]:]
		}
	}
	return "", "", s
}

// 느슨한 본문 표기: 롬8장28-30절 · 롬 8.28-30 · 로마서8 · 시편23편 · 롬 8:28-30
var reLooseRef = func() *regexp.Regexp {
	var names []string
	seen := map[string]bool{}
	for _, b := range books {
		for _, n := range append([]string{b.Name, b.Abbr}, b.alias...) {
			if !seen[n] {
				seen[n] = true
				names = append(names, strings.ReplaceAll(regexp.QuoteMeta(n), " ", `\s?`))
			}
		}
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	return regexp.MustCompile(`(?i)(` + strings.Join(names, "|") + `)\.?\s?(\d{1,3})(?:\s?(?:장|:|\.)\s?(\d{1,3})\s?절?(?:\s?[-~–]\s?(\d{1,3})(?:\s?(?:장|:)\s?(\d{1,3}))?\s?절?)?|\s?[-~–]\s?(\d{1,3})\s?(?:장|편)|\s?(?:장|편))?`)
}()

func takeRefs(s string) (string, string) {
	var refs []string
	out := s
	for {
		locs := reLooseRef.FindAllStringSubmatchIndex(out, -1)
		found := false
		for _, m := range locs {
			if !isWordStart(out, m[0]) {
				continue
			}
			g := func(k int) string {
				if m[2*k] < 0 {
					return ""
				}
				return out[m[2*k]:m[2*k+1]]
			}
			name, c, v1, v2, c2v, cRange := g(1), g(2), g(3), g(4), g(5), g(6)
			b := lookupBook(name)
			if b == 0 {
				continue
			}
			bk := bookByN(b)
			// 한 글자 약어 뒤에 숫자 하나만 있는 것(예: '주 1')은 장 표시(장 · 편 · :)가 있어야 믿어요
			whole := out[m[0]:m[1]]
			if utf8.RuneCountInString(name) == 1 && v1 == "" && !strings.ContainsAny(whole, "장편:") {
				continue
			}
			if atoi(c) < 1 || atoi(c) > bk.Chapters {
				continue
			}
			ref := bk.Abbr + " " + c
			switch {
			case v1 != "" && v2 != "" && c2v != "":
				ref += ":" + v1 + "-" + v2 + ":" + c2v
			case v1 != "" && v2 != "":
				ref += ":" + v1 + "-" + v2
			case v1 != "":
				ref += ":" + v1
			case cRange != "":
				ref += "-" + cRange + "장"
			}
			refs = append(refs, ref)
			out = out[:m[0]] + " " + out[m[1]:]
			found = true
			break
		}
		if !found {
			break
		}
	}
	return strings.Join(refs, "; "), out
}

func cleanTitle(s string) string {
	s = reNoise.ReplaceAllString(s, " ")
	s = reSpace.ReplaceAllString(s, " ")
	for i := 0; i < 3; i++ {
		s = reTrimSep.ReplaceAllString(s, "")
	}
	return strings.TrimSpace(s)
}

// parseLoose: 한 줄(또는 파일 이름) 전체에서 날짜 · 종류 · 본문 · 제목을 골라내요
func parseLoose(s string, yearHint int) ImportRow {
	src := strings.TrimSpace(s)
	t := composeHangul(src)
	t = strings.NewReplacer("_", " ", "　", " ", "（", "(", "）", ")").Replace(t)
	r := ImportRow{Src: src}
	// 본문을 먼저 떼어 내요 ("엡1장3-14절"의 3-14 를 날짜로 읽지 않게)
	r.PassageText, t = takeRefs(t)
	r.Date, t = takeDate(t, yearHint)
	r.Kind, r.KindOther, t = takeKind(t)
	r.Season, t = takeSeason(t)
	r.Title = cleanTitle(t)
	return r
}

// ---------------------------------------------------------------- 표 (칸으로 나뉜 줄들)

var headerKeys = []struct {
	field string
	words []string
}{
	{"date", []string{"날짜", "일자", "설교일", "date", "일시"}},
	{"time", []string{"시간", "time"}},
	{"kind", []string{"종류", "구분", "예배", "집회", "kind"}},
	{"ref", []string{"본문", "성경", "말씀", "구절", "passage", "text"}},
	{"title", []string{"제목", "설교제목", "title"}},
	{"place", []string{"장소", "교회", "설교장소", "place"}},
	{"summary", []string{"요약", "내용", "메모", "비고", "summary", "note"}},
	{"season", []string{"절기"}},
	{"series", []string{"시리즈", "강해"}},
	{"status", []string{"상태"}},
}

func headerField(cell string) string {
	c := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(cell), " ", ""))
	c = strings.Trim(c, "·*:()")
	if c == "" || len([]rune(c)) > 10 {
		return ""
	}
	for _, h := range headerKeys {
		for _, w := range h.words {
			if strings.Contains(c, w) {
				return h.field
			}
		}
	}
	return ""
}

func normKind(cell string) (string, string) {
	c := strings.TrimSpace(cell)
	if c == "" {
		return "", ""
	}
	if validKinds[c] {
		return c, ""
	}
	if k, o, _ := takeKind(c); k != "" {
		return k, o
	}
	return "그 외", c
}

func normDate(cell string) string {
	c := strings.TrimSpace(cell)
	if c == "" {
		return ""
	}
	// 엑셀 날짜 숫자 (1900 날짜 체계)
	if f, err := strconv.ParseFloat(c, 64); err == nil && f > 18000 && f < 80000 {
		return time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local).AddDate(0, 0, int(f)).Format("2006-01-02")
	}
	d, _ := takeDate(c, 0)
	return d
}

// rowsFromCells: 칸으로 나뉜 줄들 → 등록할 줄들 (첫 줄이 머리글이면 그대로 따르고, 아니면 칸 모양으로 짐작)
func rowsFromCells(cells [][]string) []ImportRow {
	var out []ImportRow
	if len(cells) == 0 {
		return out
	}
	cols := map[int]string{}
	hits := 0
	for i, c := range cells[0] {
		if f := headerField(c); f != "" {
			cols[i] = f
			hits++
		}
	}
	start := 0
	if hits >= 2 {
		start = 1
	} else {
		cols = nil
	}
	for _, row := range cells[start:] {
		empty := true
		for _, c := range row {
			if strings.TrimSpace(c) != "" {
				empty = false
			}
		}
		if empty {
			continue
		}
		// 서식 파일의 보기 줄은 건너뛰어요
		if strings.Contains(strings.Join(row, " "), "보기 줄") {
			continue
		}
		var parts []string
		for _, c := range row {
			if strings.TrimSpace(c) != "" {
				parts = append(parts, strings.TrimSpace(c))
			}
		}
		src := strings.Join(parts, " | ")
		var r ImportRow
		if cols != nil {
			r.Src = src
			for i, c := range row {
				c = strings.TrimSpace(composeHangul(c))
				switch cols[i] {
				case "date":
					r.Date = normDate(c)
					if r.Date == "" && c != "" {
						r.Warn = "날짜를 알아보지 못했어요: " + c
					}
				case "time":
					if m := reTimeCell.FindStringSubmatch(c); m != nil {
						r.Time = fmt.Sprintf("%02d:%s", atoi(m[1]), m[2])
					}
				case "kind":
					r.Kind, r.KindOther = normKind(c)
				case "ref":
					r.PassageText = c
				case "title":
					r.Title = c
				case "place":
					r.Place = c
				case "summary":
					r.Summary = c
				case "season":
					r.Season = c
				case "series":
					r.Series = c
				case "status":
					if validStatus[c] {
						r.Status = c
					}
				}
			}
		} else {
			// 머리글이 없으면 칸마다 모양을 봐요: 날짜 · 종류 · 본문 · 시간, 나머지는 차례대로 제목 → 장소 → 요약
			r.Src = src
			var rest []string
			for _, c := range row {
				c = strings.TrimSpace(composeHangul(c))
				if c == "" {
					continue
				}
				switch {
				case r.Date == "" && normDate(c) != "" && len([]rune(c)) <= 16:
					r.Date = normDate(c)
				case r.Time == "" && reTimeCell.MatchString(c):
					m := reTimeCell.FindStringSubmatch(c)
					r.Time = fmt.Sprintf("%02d:%s", atoi(m[1]), m[2])
				case r.Kind == "" && len([]rune(c)) <= 10 && kindOnly(c):
					r.Kind, r.KindOther = normKind(c)
				case r.PassageText == "" && refOnly(c):
					r.PassageText = c
				default:
					rest = append(rest, c)
				}
			}
			if len(row) == 1 || (r.Date == "" && r.PassageText == "" && len(rest) == 1) {
				// 칸이 하나뿐이면 한 줄을 통째로 읽어요
				lr := parseLoose(strings.Join(row, " "), 0)
				lr.Src = src
				out = append(out, lr)
				continue
			}
			if len(rest) > 0 {
				r.Title = rest[0]
			}
			if len(rest) > 1 {
				r.Place = rest[1]
			}
			if len(rest) > 2 {
				r.Summary = strings.Join(rest[2:], " / ")
			}
		}
		out = append(out, r)
	}
	return out
}

func kindOnly(c string) bool {
	k, _, rest := takeKind(c)
	return k != "" && strings.TrimSpace(rest) == ""
}

func refOnly(c string) bool {
	ps := parsePassages(c)
	if len(ps) == 0 {
		return false
	}
	for _, p := range ps {
		if !p.valid() {
			return false
		}
	}
	return true
}

// splitPasted: 붙여 넣은 글 → 칸들. 탭이 있으면 탭, 아니면 | 또는 " / ", 그것도 없으면 한 줄 통째로.
func splitPasted(text string) [][]string {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(text, "\n")
	sep := ""
	for _, l := range lines {
		switch {
		case strings.Contains(l, "\t"):
			sep = "\t"
		case sep == "" && strings.Contains(l, "|"):
			sep = "|"
		case sep == "" && strings.Contains(l, " / "):
			sep = " / "
		}
		if sep == "\t" {
			break
		}
	}
	var out [][]string
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if sep == "" {
			out = append(out, []string{l})
		} else {
			out = append(out, strings.Split(l, sep))
		}
	}
	return out
}

// ---------------------------------------------------------------- 엑셀 (.xlsx) · CSV 읽기

func readXLSX(b []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, errors.New("엑셀 파일을 열지 못했어요")
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	read := func(name string) ([]byte, error) {
		f := files[name]
		if f == nil {
			return nil, os.ErrNotExist
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, 64<<20))
	}
	// 공유 글자
	var shared []string
	if sb, err := read("xl/sharedStrings.xml"); err == nil {
		var sst struct {
			SI []struct {
				T string `xml:"t"`
				R []struct {
					T string `xml:"t"`
				} `xml:"r"`
			} `xml:"si"`
		}
		xml.Unmarshal(sb, &sst)
		for _, si := range sst.SI {
			t := si.T
			for _, r := range si.R {
				t += r.T
			}
			shared = append(shared, t)
		}
	}
	// 첫 번째 시트 찾기
	sheet := "xl/worksheets/sheet1.xml"
	if wb, err := read("xl/workbook.xml"); err == nil {
		var w struct {
			Sheets []struct {
				RID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
			} `xml:"sheets>sheet"`
		}
		xml.Unmarshal(wb, &w)
		if rels, err := read("xl/_rels/workbook.xml.rels"); err == nil && len(w.Sheets) > 0 {
			var rs struct {
				R []struct {
					ID     string `xml:"Id,attr"`
					Target string `xml:"Target,attr"`
				} `xml:"Relationship"`
			}
			xml.Unmarshal(rels, &rs)
			for _, r := range rs.R {
				if r.ID == w.Sheets[0].RID {
					t := strings.TrimPrefix(r.Target, "/")
					if !strings.HasPrefix(t, "xl/") {
						t = "xl/" + t
					}
					sheet = t
				}
			}
		}
	}
	sb, err := read(sheet)
	if err != nil {
		return nil, errors.New("엑셀 파일에서 시트를 찾지 못했어요")
	}
	var ws struct {
		Rows []struct {
			C []struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				V  string `xml:"v"`
				IS struct {
					T string `xml:"t"`
					R []struct {
						T string `xml:"t"`
					} `xml:"r"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(sb, &ws); err != nil {
		return nil, errors.New("엑셀 파일을 읽지 못했어요")
	}
	var out [][]string
	for _, row := range ws.Rows {
		var cells []string
		for _, c := range row.C {
			col := colIndex(c.R)
			for len(cells) < col {
				cells = append(cells, "")
			}
			v := c.V
			switch c.T {
			case "s":
				if i, err := strconv.Atoi(c.V); err == nil && i >= 0 && i < len(shared) {
					v = shared[i]
				}
			case "inlineStr":
				v = c.IS.T
				for _, r := range c.IS.R {
					v += r.T
				}
			}
			if col >= 0 && col < len(cells) {
				cells[col] = v
			} else {
				cells = append(cells, v)
			}
		}
		out = append(out, cells)
	}
	return out, nil
}

// colIndex: "C12" → 2
func colIndex(ref string) int {
	n := 0
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			n = n*26 + int(r-'A'+1)
		} else {
			break
		}
	}
	return n - 1
}

func readCSV(b []byte) ([][]string, error) {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(b) {
		return nil, errors.New("이 CSV 파일은 한글 글자 방식이 달라서 읽을 수 없어요. 엑셀에서 ‘Excel 통합 문서(.xlsx)’로 저장해서 넣어 주세요")
	}
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	return r.ReadAll()
}

// ---------------------------------------------------------------- 원고 파일 폴더

var manuscriptExt = map[string]bool{".hwp": true, ".hwpx": true, ".doc": true, ".docx": true, ".pdf": true, ".pages": true, ".txt": true,
	".md": true, ".rtf": true, ".odt": true, ".key": true, ".ppt": true, ".pptx": true, ".m4a": true, ".mp3": true, ".wav": true, ".mp4": true, ".mov": true}

var reYearDir = regexp.MustCompile(`(?:^|\D)((?:19|20)\d{2})(?:\D|$)`)

func rowsFromFolder(root string, recursive bool) ([]ImportRow, error) {
	if !dirExists(root) {
		return nil, errors.New("폴더를 찾지 못했어요")
	}
	var out []ImportRow
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || !recursive) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "~$") || !manuscriptExt[ext] {
			return nil
		}
		if len(out) >= 3000 {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(root, p)
		// 폴더 이름의 해(예: 2019년 설교/0310 …)를 날짜 짐작에 써요
		hint := 0
		for _, part := range strings.Split(filepath.Dir(rel), string(filepath.Separator)) {
			if m := reYearDir.FindStringSubmatch(part); m != nil {
				hint = atoi(m[1])
			}
		}
		if hint == 0 {
			if m := reYearDir.FindStringSubmatch(filepath.Base(root)); m != nil {
				hint = atoi(m[1])
			}
		}
		name := composeHangul(strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())))
		r := parseLoose(name, hint)
		r.Src = composeHangul(rel)
		r.File = p
		r.FileName = composeHangul(d.Name())
		out = append(out, r)
		return nil
	})
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			if out[i].Date == "" || out[j].Date == "" {
				return out[j].Date == "" // 날짜 없는 파일은 맨 아래로
			}
			return out[i].Date < out[j].Date
		}
		return out[i].Src < out[j].Src
	})
	return out, err
}

// ---------------------------------------------------------------- 마무리: 본문 알아보기 · 겹침 표시

func (a *App) finishRows(rows []ImportRow) []ImportRow {
	home := ""
	if v, ok := a.Settings().UI["homePlace"].(string); ok {
		home = v
	}
	a.store.mu.Lock()
	existing := make([]*Sermon, len(a.store.data.Sermons))
	copy(existing, a.store.data.Sermons)
	a.store.mu.Unlock()
	if home == "" {
		// 기본 장소를 정하지 않았으면 가장 많이 설교한 곳을 써요
		count := map[string]int{}
		for _, s := range existing {
			if s.Place != "" {
				count[s.Place]++
			}
		}
		for p, n := range count {
			if n > count[home] {
				home = p
			}
		}
	}
	seen := map[string]bool{}
	for i := range rows {
		r := &rows[i]
		if r.Place == "" {
			r.Place = home
		}
		if r.Kind == "" {
			r.Kind = "주일"
			if d, err := time.Parse("2006-01-02", r.Date); err == nil {
				switch d.Weekday() {
				case time.Wednesday:
					r.Kind = "수요"
				case time.Friday:
					r.Kind = "금요"
				case time.Sunday:
				default:
					r.Kind = "새벽"
				}
			}
		}
		if r.Status == "" {
			r.Status = "설교함"
		}
		ps := parsePassages(r.PassageText)
		r.RefOK = len(ps) > 0
		for _, p := range ps {
			if !p.valid() {
				r.RefOK = false
			}
		}
		r.Ref = refsText(ps)
		if r.Date == "" && r.Warn == "" {
			r.Warn = "날짜가 없어요"
		}
		// 이미 있는 기록과 겹치는지: 같은 날 + (같은 본문 또는 같은 제목)
		key := r.Date + "|" + r.Ref + "|" + r.Title
		if seen[key] {
			r.Dup = "이 목록에 같은 줄이 또 있어요"
		}
		seen[key] = true
		for _, s := range existing {
			if s.Date != r.Date || r.Date == "" {
				continue
			}
			same := r.Title != "" && strings.TrimSpace(s.Title) == r.Title
			for _, p := range ps {
				for _, q := range s.Passages {
					if overlaps(p, q) {
						same = true
					}
				}
			}
			if same {
				r.Dup = "이미 기록에 있어요: " + firstNonEmpty(s.Title, refsText(s.Passages))
				break
			}
		}
	}
	return rows
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// ---------------------------------------------------------------- HTTP

func (a *App) handleImportParse(w http.ResponseWriter, r *http.Request) {
	var rows []ImportRow
	ct := r.Header.Get("content-type")
	if strings.HasPrefix(ct, "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
		f, h, err := r.FormFile("file")
		if err != nil {
			jsonError(w, 400, "파일을 받지 못했어요")
			return
		}
		b, _ := io.ReadAll(f)
		f.Close()
		var cells [][]string
		switch strings.ToLower(filepath.Ext(h.Filename)) {
		case ".xlsx", ".xlsm":
			cells, err = readXLSX(b)
		case ".csv", ".txt", ".tsv":
			if strings.HasSuffix(strings.ToLower(h.Filename), ".csv") {
				cells, err = readCSV(b)
			} else if utf8.Valid(b) {
				cells = splitPasted(string(b))
			} else {
				err = errors.New("글자 방식이 달라서 읽을 수 없어요")
			}
		case ".xls":
			err = errors.New("오래된 엑셀 파일(.xls)은 읽을 수 없어요. 엑셀에서 ‘Excel 통합 문서(.xlsx)’로 다시 저장해서 넣어 주세요")
		default:
			err = errors.New("엑셀 파일(.xlsx)이나 CSV 파일을 넣어 주세요")
		}
		if err != nil {
			jsonError(w, 400, err.Error())
			return
		}
		rows = rowsFromCells(cells)
	} else {
		var in struct {
			Mode      string `json:"mode"` // text | folder
			Text      string `json:"text"`
			Path      string `json:"path"`
			Recursive bool   `json:"recursive"`
		}
		if err := readJSONBig(r, &in); err != nil {
			jsonError(w, 400, "보낸 내용을 읽지 못했어요")
			return
		}
		switch in.Mode {
		case "folder":
			var err error
			rows, err = rowsFromFolder(in.Path, in.Recursive)
			if err != nil && rows == nil {
				jsonError(w, 400, err.Error())
				return
			}
		default:
			rows = rowsFromCells(splitPasted(in.Text))
		}
	}
	if len(rows) == 0 {
		jsonError(w, 400, "읽을 설교를 찾지 못했어요")
		return
	}
	if len(rows) > 3000 {
		rows = rows[:3000]
	}
	writeJSON(w, map[string]any{"rows": a.finishRows(rows)})
}

func (a *App) handleImportCommit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Rows   []ImportRow `json:"rows"`
		Attach bool        `json:"attach"`
	}
	if err := readJSONBig(r, &in); err != nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	if len(in.Rows) == 0 {
		jsonError(w, 400, "등록할 설교가 없어요")
		return
	}
	for i, x := range in.Rows {
		if _, err := time.Parse("2006-01-02", x.Date); err != nil {
			jsonError(w, 400, fmt.Sprintf("%d번째 줄의 날짜를 확인해 주세요", i+1))
			return
		}
	}
	s := a.store
	batch := "b" + randID(4)
	// 먼저 지금 상태를 백업해 둬요
	s.mu.Lock()
	cur, _ := json.MarshalIndent(s.data, "", " ")
	s.mu.Unlock()
	if _, err := s.writeBackup("before-import", cur); err != nil {
		jsonError(w, 500, "등록 전에 백업하지 못해서 멈췄어요")
		return
	}
	// 시리즈 이름 → 아이디 (없으면 새로 만들어요)
	s.mu.Lock()
	seriesID := map[string]string{}
	for _, x := range s.data.Series {
		seriesID[x.Name] = x.ID
	}
	s.mu.Unlock()
	type pending struct {
		x    *Sermon
		file string
	}
	var list []pending
	now := nowStamp()
	for _, row := range in.Rows {
		x := &Sermon{ID: "s" + randID(6), Date: row.Date, Time: row.Time, Kind: row.Kind, KindOther: row.KindOther, Season: row.Season,
			Title: strings.TrimSpace(row.Title), PassageText: strings.TrimSpace(row.PassageText), Summary: row.Summary, Place: strings.TrimSpace(row.Place),
			Status: row.Status, Points: []string{}, Confessions: []ConfLink{}, Illus: []string{}, Files: []FileRef{},
			Extra: map[string]string{"import": batch}, Created: now, Updated: now}
		if !validKinds[x.Kind] {
			x.Kind = "그 외"
		}
		if !validStatus[x.Status] {
			x.Status = "설교함"
		}
		x.Passages = parsePassages(x.PassageText)
		if n := strings.TrimSpace(row.Series); n != "" {
			id := seriesID[n]
			if id == "" {
				id = "r" + randID(5)
				seriesID[n] = id
				s.mu.Lock()
				s.data.Series = append(s.data.Series, &Series{ID: id, Name: n, Kind: x.Kind, Color: "#8A90A6", Created: now})
				s.mu.Unlock()
			}
			x.SeriesID = id
		}
		list = append(list, pending{x, row.File})
	}
	// 원고 파일 복사 (잠그지 않고)
	copied := 0
	var failed []string
	if in.Attach {
		for _, p := range list {
			if p.file == "" {
				continue
			}
			if f, err := a.copyIntoFiles(p.x.ID, p.file); err == nil {
				p.x.Files = append(p.x.Files, f)
				copied++
			} else {
				failed = append(failed, filepath.Base(p.file))
			}
		}
	}
	s.mu.Lock()
	for _, p := range list {
		s.data.Sermons = append(s.data.Sermons, p.x)
	}
	err := s.save()
	rev := s.rev
	s.mu.Unlock()
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "batch": batch, "count": len(list), "files": copied, "failed": failed, "rev": rev})
}

func (a *App) copyIntoFiles(sid, src string) (FileRef, error) {
	in, err := os.Open(src)
	if err != nil {
		return FileRef{}, err
	}
	defer in.Close()
	name := safeName(composeHangul(filepath.Base(src)))
	id := randID(4)
	dir := filepath.Join(a.store.filesDir(), sid)
	os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, id+"_"+name)
	out, err := os.Create(p)
	if err != nil {
		return FileRef{}, err
	}
	n, err := io.Copy(out, in)
	out.Close()
	if err != nil {
		os.Remove(p)
		return FileRef{}, err
	}
	return FileRef{ID: id, Name: name, Size: n, Added: nowStamp()}, nil
}

func (a *App) handleImportUndo(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Batch string `json:"batch"`
	}
	readJSON(r, &in)
	if in.Batch == "" {
		jsonError(w, 400, "되돌릴 등록을 찾지 못했어요")
		return
	}
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	keep := s.data.Sermons[:0]
	n := 0
	for _, x := range s.data.Sermons {
		if x.Extra != nil && x.Extra["import"] == in.Batch {
			os.RemoveAll(filepath.Join(s.filesDir(), x.ID))
			n++
			continue
		}
		keep = append(keep, x)
	}
	s.data.Sermons = keep
	if err := s.save(); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "count": n, "rev": s.rev})
}

// 서식 파일: 엑셀에서 채워 넣을 수 있게 머리글과 보기 두 줄
func (a *App) handleImportTemplate(w http.ResponseWriter, r *http.Request) {
	rows := [][]string{
		{"날짜", "종류", "본문", "제목", "장소", "시리즈", "요약 · 메모"},
		{"2019-03-10", "주일", "롬 8:28-30", "하나님의 영원한 작정", "본 교회", "로마서 강해", "(이 보기 줄은 지우고 쓰세요)"},
		{"2019-03-13", "수요", "시편 23편", "여호와는 나의 목자시니", "본 교회", "", "(이 보기 줄은 지우고 쓰세요)"},
	}
	b, err := writeXLSX(rows, "설교 목록", []int{13, 10, 20, 34, 16, 18, 40})
	if err != nil {
		jsonError(w, 500, "서식 파일을 만들지 못했어요")
		return
	}
	p, err := uniquePath(exportDir(), "SermonROAD 한꺼번에 등록 서식.xlsx")
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		jsonError(w, 500, "파일을 저장하지 못했어요")
		return
	}
	revealPaths([]string{p})
	writeJSON(w, map[string]any{"ok": true, "path": p})
}
