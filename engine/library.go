package main

// 서재 (설교 도우미의 참고 자료)
//   사용자가 고른 폴더 안의 PDF 를 읽어 쪽마다 글자를 보관하고(Support/library/<id>.json),
//   쪽마다 성경 본문 표기(롬 8:28, 8장 28절 …)를 찾아 둬요. 본문을 정하면 그 본문을 다룬 쪽들을 찾아 줘요.
//
//   글자 읽기: 맥에서는 앱 안의 작은 도구(sermonroad-pdftext)가 PDFKit 과 맥의 글자 인식(Vision)으로 읽어요.
//   (스캔한 PDF 도 읽을 수 있어요.) 시험용 리눅스에서는 pdftotext 를 써요.

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type LibDoc struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Name   string `json:"name"`
	Rel    string `json:"rel"` // 폴더 안의 위치
	Size   int64  `json:"size"`
	Mod    int64  `json:"mod"`
	Pages  int    `json:"pages"`
	OCR    int    `json:"ocr"`    // 글자 인식으로 읽은 쪽 수
	Book   int    `json:"book"`   // 주로 다루는 책 (0 = 여러 책)
	Status string `json:"status"` // waiting | reading | ready | error
	Err    string `json:"err,omitempty"`
	Prog   string `json:"prog,omitempty"` // 읽는 중: "34/220"
	Refs   int    `json:"refs"`           // 찾은 본문 표기 수
	BookOv int    `json:"bookOverride"`   // 사용자가 정한 책

	text []string     // 쪽마다 글자
	refs [][]VerseRef // 쪽마다 본문 표기
}

type pdfCache struct {
	Path  string   `json:"path"`
	Size  int64    `json:"size"`
	Mod   int64    `json:"mod"`
	Pages []string `json:"pages"`
	OCR   []int    `json:"ocr"`
	Title string   `json:"title"`
}

type Library struct {
	mu      sync.Mutex
	support string
	folder  string
	docs    []*LibDoc
	byID    map[string]*LibDoc
	busy    bool
	wake    chan struct{}
	rev     int64
	err     string
	books   map[string]int // 사용자가 정한 '이 자료는 ○○서' (id → 책)
}

func NewLibrary(support string) *Library {
	l := &Library{support: support, byID: map[string]*LibDoc{}, wake: make(chan struct{}, 1), books: map[string]int{}}
	os.MkdirAll(l.cacheDir(), 0o755)
	if b, err := os.ReadFile(filepath.Join(l.cacheDir(), "books.json")); err == nil {
		json.Unmarshal(b, &l.books)
	}
	go l.worker()
	return l
}

func (l *Library) cacheDir() string { return filepath.Join(l.support, "library") }

func docID(p string) string {
	h := sha1.Sum([]byte(p))
	return hex.EncodeToString(h[:])[:14]
}

// SetFolder: 서재 폴더를 바꾸고 다시 훑어요
func (l *Library) SetFolder(folder string) {
	l.mu.Lock()
	l.folder = folder
	l.mu.Unlock()
	l.Scan()
}

// Scan: 폴더 안의 PDF 목록을 다시 만들고, 읽지 않은(또는 바뀐) PDF 를 읽을 차례에 넣어요
func (l *Library) Scan() {
	l.mu.Lock()
	folder := l.folder
	l.mu.Unlock()
	var found []*LibDoc
	errMsg := ""
	if folder != "" {
		if !dirExists(folder) {
			errMsg = "서재 폴더를 찾지 못했어요. 옮겼거나 이름이 바뀌었는지 확인해 주세요"
		} else {
			filepath.WalkDir(folder, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					if p != folder && strings.HasPrefix(d.Name(), ".") {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasPrefix(d.Name(), ".") || !strings.EqualFold(filepath.Ext(d.Name()), ".pdf") {
					return nil
				}
				info, err := d.Info()
				if err != nil {
					return nil
				}
				rel, _ := filepath.Rel(folder, p)
				found = append(found, &LibDoc{ID: docID(p), Path: p, Name: composeHangul(strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))),
					Rel: composeHangul(rel), Size: info.Size(), Mod: info.ModTime().Unix(), Status: "waiting"})
				return nil
			})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Rel < found[j].Rel })
	l.mu.Lock()
	byID := map[string]*LibDoc{}
	for _, d := range found {
		if old := l.byID[d.ID]; old != nil && old.Size == d.Size && old.Mod == d.Mod && old.Status != "error" {
			d = old
		} else if c := l.loadCache(d); c != nil {
			l.applyCache(d, c)
		}
		byID[d.ID] = d
	}
	docs := make([]*LibDoc, 0, len(found))
	for _, d := range found {
		docs = append(docs, byID[d.ID])
	}
	l.docs, l.byID, l.err = docs, byID, errMsg
	l.rev = time.Now().UnixMilli()
	l.mu.Unlock()
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

func (l *Library) cachePath(id string) string { return filepath.Join(l.cacheDir(), id+".json") }

func (l *Library) loadCache(d *LibDoc) *pdfCache {
	b, err := os.ReadFile(l.cachePath(d.ID))
	if err != nil {
		return nil
	}
	var c pdfCache
	if json.Unmarshal(b, &c) != nil || c.Size != d.Size || c.Mod != d.Mod {
		return nil
	}
	return &c
}

// applyCache: 읽어 둔 글자로 본문 표기 찾기 (l.mu 를 잡은 채로 불러도 돼요)
func (l *Library) applyCache(d *LibDoc, c *pdfCache) {
	d.text = c.Pages
	d.Pages = len(c.Pages)
	d.OCR = len(c.OCR)
	d.BookOv = l.books[d.ID]
	d.Book = d.BookOv
	if d.Book == 0 {
		d.Book = guessBook(d.Name+" "+c.Title, c.Pages)
	}
	d.refs = make([][]VerseRef, len(c.Pages))
	cur := 0
	n := 0
	for i, t := range c.Pages {
		var r []VerseRef
		r, cur = findRefs(t, d.Book, cur)
		d.refs[i] = r
		n += len(r)
	}
	d.Refs = n
	d.Status, d.Err, d.Prog = "ready", "", ""
}

func (l *Library) worker() {
	defer recoverLog("library worker")
	for range l.wake {
		for {
			l.mu.Lock()
			var next *LibDoc
			for _, d := range l.docs {
				if d.Status == "waiting" {
					next = d
					break
				}
			}
			if next != nil {
				next.Status = "reading"
				l.busy = true
			} else {
				l.busy = false
			}
			l.rev = time.Now().UnixMilli()
			l.mu.Unlock()
			if next == nil {
				break
			}
			c, err := l.extract(next)
			l.mu.Lock()
			if err != nil {
				next.Status, next.Err, next.Prog = "error", err.Error(), ""
				log.Printf("서재: %s 읽기 실패: %v", next.Name, err)
			} else {
				l.applyCache(next, c)
				log.Printf("서재: %s 읽음 (%d쪽, 글자 인식 %d쪽, 본문 표기 %d곳, 주된 책 %d)", next.Name, next.Pages, next.OCR, next.Refs, next.Book)
			}
			l.rev = time.Now().UnixMilli()
			l.mu.Unlock()
		}
	}
}

var rePageProg = regexp.MustCompile(`^PAGE (\d+)/(\d+)`)

// extract: PDF 글자를 읽어 캐시에 저장해요
func (l *Library) extract(d *LibDoc) (*pdfCache, error) {
	out := l.cachePath(d.ID) + ".part"
	defer os.Remove(out)
	var pages []string
	var ocr []int
	title := ""
	switch {
	case env("PDFTEXT") != "": // 시험용: 이 명령이 <PDF> <결과.json> 을 받아요
		if err := l.runPDFHelper(d, env("PDFTEXT"), []string{d.Path, out}); err != nil {
			return nil, err
		}
	case runtime.GOOS == "darwin" && env("APP") != "":
		bin := filepath.Join(env("APP"), "Contents", "Resources", "sermonroad-pdftext")
		if !fileExists(bin) {
			return nil, errors.New("앱을 찾지 못해서 PDF를 읽을 수 없어요. 앱을 다시 실행해 주세요")
		}
		if err := l.runPDFHelper(d, bin, []string{d.Path, out}); err != nil {
			return nil, err
		}
	default:
		pt, err := exec.LookPath("pdftotext")
		if err != nil {
			return nil, errors.New("이 컴퓨터에서는 PDF 글자를 읽을 수 없어요")
		}
		b, err := exec.Command(pt, "-enc", "UTF-8", d.Path, "-").Output()
		if err != nil {
			return nil, errors.New("PDF를 읽지 못했어요")
		}
		pages = strings.Split(string(b), "\f")
		if len(pages) > 0 && strings.TrimSpace(pages[len(pages)-1]) == "" {
			pages = pages[:len(pages)-1]
		}
		goto done
	}
	{
		b, err := os.ReadFile(out)
		if err != nil {
			return nil, errors.New("PDF 글자를 읽지 못했어요")
		}
		var r struct {
			Pages []string `json:"pages"`
			OCR   []int    `json:"ocr"`
			Title string   `json:"title"`
		}
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, errors.New("PDF 글자를 읽지 못했어요")
		}
		pages, ocr, title = r.Pages, r.OCR, r.Title
	}
done:
	for i := range pages {
		pages[i] = cleanText(pages[i])
	}
	c := &pdfCache{Path: d.Path, Size: d.Size, Mod: d.Mod, Pages: pages, OCR: ocr, Title: title}
	if ocr == nil {
		c.OCR = []int{}
	}
	b, _ := json.Marshal(c)
	writeFileAtomic(l.cachePath(d.ID), b, 0o644)
	return c, nil
}

func (l *Library) runPDFHelper(d *LibDoc, bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	hideWindow(cmd)
	stdout, _ := cmd.StdoutPipe()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("PDF를 읽지 못했어요: %v", err)
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-done:
		case <-time.After(90 * time.Minute):
			cmd.Process.Kill()
		}
	}()
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		if m := rePageProg.FindStringSubmatch(sc.Text()); m != nil {
			l.mu.Lock()
			d.Prog = m[1] + "/" + m[2]
			l.rev = time.Now().UnixMilli()
			l.mu.Unlock()
		}
	}
	err := cmd.Wait()
	close(done)
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "PDF를 읽지 못했어요"
		}
		return errors.New(msg)
	}
	return nil
}

var reSpaces = regexp.MustCompile(`[ \t\x{00a0}]+`)
var reManyNL = regexp.MustCompile(`\n{3,}`)

func cleanText(s string) string {
	s = composeHangul(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = reSpaces.ReplaceAllString(s, " ")
	s = reManyNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func (l *Library) Snapshot() map[string]any {
	l.mu.Lock()
	defer l.mu.Unlock()
	pages, ready, ocr := 0, 0, 0
	for _, d := range l.docs {
		pages += d.Pages
		ocr += d.OCR
		if d.Status == "ready" {
			ready++
		}
	}
	return map[string]any{"folder": l.folder, "docs": l.docs, "busy": l.busy, "rev": l.rev, "error": l.err,
		"total": len(l.docs), "ready": ready, "pages": pages, "ocr": ocr}
}

func (l *Library) SetBook(id string, book int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	d := l.byID[id]
	if d == nil {
		return
	}
	if book == 0 {
		delete(l.books, id)
	} else {
		l.books[id] = book
	}
	b, _ := json.Marshal(l.books)
	writeFileAtomic(filepath.Join(l.cacheDir(), "books.json"), b, 0o644)
	if d.Status == "ready" {
		l.applyCache(d, &pdfCache{Pages: d.text, OCR: make([]int, d.OCR)})
	}
	l.rev = time.Now().UnixMilli()
}

// ---------------------------------------------------------------- 본문으로 찾기

type LibHit struct {
	Doc     string  `json:"doc"`
	Name    string  `json:"name"`
	Book    int     `json:"book"`
	From    int     `json:"from"` // 1부터 세는 PDF 쪽
	To      int     `json:"to"`
	Hits    int     `json:"hits"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
}

// Find: 본문(여러 곳 가능)을 다룬 쪽들 — 문서마다 가장 알맞은 구간 1~2개
func (l *Library) Find(ps []Passage) []LibHit {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []LibHit
	for _, d := range l.docs {
		if d.Status != "ready" {
			continue
		}
		scores := make([]float64, len(d.refs))
		hits := make([]int, len(d.refs))
		for i, refs := range d.refs {
			for _, r := range refs {
				for _, p := range ps {
					if !p.valid() || r.Book != p.Book {
						continue
					}
					c2 := p.C2
					if c2 == 0 {
						c2 = p.C1
					}
					if r.C < p.C1 || r.C > c2 {
						continue
					}
					if r.V1 == 0 { // 장만 (장 제목)
						scores[i] += 1.5
						continue
					}
					q := Passage{Book: r.Book, C1: r.C, V1: r.V1, C2: r.C, V2: r.V2}
					if overlaps(q, p) {
						scores[i] += 3
						hits[i]++
					} else {
						scores[i] += 0.4
					}
				}
			}
		}
		// 이어진 쪽 묶기 (사이에 한 쪽이 비어도 이어 봐요)
		type span struct {
			from, to, hits int
			score          float64
		}
		var spans []span
		cur := span{from: -1}
		gap := 0
		for i, sc := range scores {
			if sc > 0 {
				if cur.from < 0 {
					cur = span{from: i, to: i}
				}
				cur.to = i
				cur.score += sc
				cur.hits += hits[i]
				gap = 0
			} else if cur.from >= 0 {
				gap++
				if gap > 1 {
					spans = append(spans, cur)
					cur = span{from: -1}
					gap = 0
				}
			}
		}
		if cur.from >= 0 {
			spans = append(spans, cur)
		}
		// 점수가 너무 낮은 구간(본문 표기 하나 스치고 지나간 곳)은 빼요
		sort.Slice(spans, func(i, j int) bool { return spans[i].score > spans[j].score })
		for k, sp := range spans {
			if k >= 2 || sp.score < 3 || (k > 0 && sp.score < spans[0].score*0.35) {
				break
			}
			// 너무 긴 구간은 점수가 높은 쪽 둘레 12쪽까지만
			if sp.to-sp.from > 11 {
				best := sp.from
				for i := sp.from; i <= sp.to; i++ {
					if scores[i] > scores[best] {
						best = i
					}
				}
				sp.from = max(sp.from, best-3)
				sp.to = min(sp.to, sp.from+11)
			}
			out = append(out, LibHit{Doc: d.ID, Name: d.Name, Book: d.Book, From: sp.from + 1, To: sp.to + 1, Hits: sp.hits,
				Score: sp.score, Snippet: snippetFor(d, bestPage(scores, sp.from, sp.to), sp.to, ps)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func bestPage(scores []float64, from, to int) int {
	best := from
	for i := from; i <= to && i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}
	return best
}

// snippetFor: 그 쪽에서 고른 본문과 맞는 표기 둘레의 글을 보여 줘요
func snippetFor(d *LibDoc, from, to int, ps []Passage) string {
	if from < len(d.text) {
		t := d.text[from]
		for _, m := range reTextRef.FindAllStringSubmatchIndex(t, -1) {
			b := lookupBook(t[m[2]:m[3]])
			for _, p := range ps {
				if b == p.Book {
					return around(t, m[0], 70, 110)
				}
			}
		}
		for _, m := range reBareVerse.FindAllStringSubmatchIndex(t, -1) {
			v := atoi(t[m[2]:m[3]])
			for _, p := range ps {
				if p.V1 > 0 && v >= p.V1 && v <= max(p.V2, p.V1) {
					return around(t, m[0], 70, 110)
				}
			}
		}
	}
	for i := from; i <= to && i < len(d.text); i++ {
		t := d.text[i]
		for _, m := range reTextRef.FindAllStringIndex(t, -1) {
			return around(t, m[0], 70, 110)
		}
		for _, m := range reBareVerse.FindAllStringIndex(t, -1) {
			return around(t, m[0], 70, 110)
		}
	}
	if from < len(d.text) {
		return around(d.text[from], 0, 0, 160)
	}
	return ""
}

func around(t string, at, before, after int) string {
	start := at
	for i := 0; i < before && start > 0; i++ {
		_, sz := utf8.DecodeLastRuneInString(t[:start])
		start -= sz
	}
	end := at
	for i := 0; i < after && end < len(t); i++ {
		_, sz := utf8.DecodeRuneInString(t[end:])
		end += sz
	}
	s := strings.Join(strings.Fields(t[start:end]), " ")
	if start > 0 {
		s = "…" + s
	}
	if end < len(t) {
		s += "…"
	}
	return s
}

// PagesText: 도우미에게 보낼 글자 — [12쪽] 표시를 붙여요
func (l *Library) PagesText(id string, from, to, limit int) (string, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	d := l.byID[id]
	if d == nil || d.Status != "ready" {
		return "", "", errors.New("자료를 찾지 못했어요. 서재를 다시 읽어 주세요")
	}
	if from < 1 {
		from = 1
	}
	if to > len(d.text) {
		to = len(d.text)
	}
	var b strings.Builder
	for i := from; i <= to; i++ {
		t := d.text[i-1]
		fmt.Fprintf(&b, "[%d쪽]\n%s\n\n", i, t)
		if b.Len() > limit {
			break
		}
	}
	s := b.String()
	if len(s) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + "\n…(길어서 줄였어요)"
	}
	return d.Name, s, nil
}

func (l *Library) doc(id string) *LibDoc {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.byID[id]
}

// ---------------------------------------------------------------- HTTP

func (a *App) handleLibrary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.lib.Snapshot())
}

func (a *App) handleLibraryFolder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path string `json:"path"`
	}
	readJSON(r, &in)
	if in.Path != "" && !dirExists(in.Path) {
		jsonError(w, 400, "폴더를 찾을 수 없어요")
		return
	}
	a.mu.Lock()
	a.settings.Library = in.Path
	a.saveSettings()
	a.mu.Unlock()
	a.lib.SetFolder(in.Path)
	writeJSON(w, a.lib.Snapshot())
}

func (a *App) handleLibraryRescan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Doc string `json:"doc"` // 이 자료만 다시 읽기
	}
	readJSON(r, &in)
	if in.Doc != "" {
		a.lib.mu.Lock()
		if d := a.lib.byID[in.Doc]; d != nil {
			os.Remove(a.lib.cachePath(d.ID))
			d.Status, d.Err = "waiting", ""
		}
		a.lib.mu.Unlock()
	}
	a.lib.Scan()
	writeJSON(w, a.lib.Snapshot())
}

func (a *App) handleLibraryBook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Doc  string `json:"doc"`
		Book int    `json:"book"`
	}
	readJSON(r, &in)
	a.lib.SetBook(in.Doc, in.Book)
	writeJSON(w, a.lib.Snapshot())
}

func (a *App) handleLibraryFind(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Ref string `json:"ref"`
	}
	readJSON(r, &in)
	ps := parsePassages(in.Ref)
	ok := false
	for _, p := range ps {
		if p.valid() {
			ok = true
		}
	}
	if !ok {
		jsonError(w, 400, "본문을 알아보지 못했어요. 예: 롬 8:28-30, 로마서 8장")
		return
	}
	writeJSON(w, map[string]any{"passages": ps, "hits": a.lib.Find(ps)})
}

func (a *App) handleLibraryPage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("doc")
	p, _ := strconv.Atoi(r.URL.Query().Get("p"))
	d := a.lib.doc(id)
	if d == nil || d.Status != "ready" || p < 1 || p > len(d.text) {
		jsonError(w, 404, "그 쪽을 찾지 못했어요")
		return
	}
	writeJSON(w, map[string]any{"doc": id, "name": d.Name, "p": p, "pages": d.Pages, "text": d.text[p-1]})
}

func (a *App) handleLibraryOpen(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Doc string `json:"doc"`
	}
	readJSON(r, &in)
	d := a.lib.doc(in.Doc)
	if d == nil {
		jsonError(w, 404, "자료를 찾지 못했어요")
		return
	}
	openPaths([]string{d.Path})
	writeJSON(w, map[string]any{"ok": true})
}
