package main

// 설교 기록 저장소
//   Support/data.json 하나에 설교 · 강해 시리즈 · 예화 · 반복 일정을 모두 담아요.
//   - 저장은 늘 '임시 파일에 쓰고 바꿔치기'라서 중간에 꺼져도 파일이 깨지지 않아요.
//   - 하루에 한 번(바뀐 것이 있을 때) 자동 백업: Support/backups/auto-YYYYMMDD-HHMMSS.json (최근 20개)
//   - '지금 백업': manual-… (최근 20개) · 되살리기 전에는 지금 상태를 before-restore-… 로 남겨요.
//   - 지운 설교는 휴지통(최근 50개)에 남아서 되살릴 수 있어요.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type FileRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Added string `json:"added"`
}

type ConfLink struct {
	Doc string `json:"doc"` // hc | wsc | wcf | dort
	N   int    `json:"n"`
}

type Sermon struct {
	ID          string            `json:"id"`
	Date        string            `json:"date"`      // 2026-11-01
	Time        string            `json:"time"`      // 11:00
	Kind        string            `json:"kind"`      // 주일 | 주일 오후 | 새벽 | 수요 | 금요 | 성찬 | 부서 | 수련회 | 그 외
	KindOther   string            `json:"kindOther"` // 그 외: 이름 · 부서: 어느 부서 · 수련회: 어떤 수련회 (적지 않아도 돼요)
	Season      string            `json:"season"`
	Title       string            `json:"title"`
	PassageText string            `json:"passageText"` // 사용자가 적은 본문 표기 그대로
	Passages    []Passage         `json:"passages"`
	BibleText   string            `json:"bibleText"` // 직접 넣은 성경 본문
	Thesis      string            `json:"thesis"`
	Points      []string          `json:"points"`
	Summary     string            `json:"summary"`
	Place       string            `json:"place"`
	Status      string            `json:"status"` // 본문 미정 | 준비 중 | 원고 완료 | 설교함
	SeriesID    string            `json:"seriesId"`
	SeriesNo    int               `json:"seriesNo"`
	Confessions []ConfLink        `json:"confessions"`
	Illus       []string          `json:"illus"`
	Refs        string            `json:"refs"` // 참고한 자료
	Files       []FileRef         `json:"files"`
	Helper      map[string]any    `json:"helper,omitempty"` // 설교 도우미 결과 (탭 → 결과)
	Meditation  string            `json:"meditation"`
	Extra       map[string]string `json:"extra,omitempty"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
}

type Series struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Color   string `json:"color"`
	Total   int    `json:"total"`   // 예정 편 수 (0 = 정하지 않음)
	NextRef string `json:"nextRef"` // 다음 본문
	Start   string `json:"start"`
	Done    bool   `json:"done"`
	Note    string `json:"note"`
	Created string `json:"created"`
}

type Illustration struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Source  string   `json:"source"`
	Tags    []string `json:"tags"`
	Created string   `json:"created"`
	Updated string   `json:"updated"`
}

type Rule struct {
	ID       string `json:"id"`
	Dow      int    `json:"dow"`   // 0 = 주일 … 6 = 토요일
	Every    int    `json:"every"` // 1 = 매주, 2 = 격주
	Time     string `json:"time"`
	Kind     string `json:"kind"`
	Place    string `json:"place"`
	SeriesID string `json:"seriesId"`
	Note     string `json:"note"`
	Active   bool   `json:"active"`
	Start    string `json:"start"` // 격주 기준일
}

type Data struct {
	V       int             `json:"v"`
	Sermons []*Sermon       `json:"sermons"`
	Series  []*Series       `json:"series"`
	Illus   []*Illustration `json:"illus"`
	Rules   []*Rule         `json:"rules"`
	Trash   []*Sermon       `json:"trash"`
}

type Store struct {
	mu       sync.Mutex
	dir      string
	data     *Data
	rev      int64
	lastAuto string // 오늘 자동 백업을 했는지 (YYYYMMDD)
}

func NewStore(support string) *Store {
	s := &Store{dir: support, data: &Data{V: 1}}
	os.MkdirAll(s.backupDir(), 0o755)
	os.MkdirAll(s.filesDir(), 0o755)
	if b, err := os.ReadFile(s.path()); err == nil {
		var d Data
		if err := json.Unmarshal(b, &d); err != nil {
			// 파일이 깨졌으면 그대로 두지 않고 따로 보관한 뒤, 가장 최근 백업에서 되살려요
			bad := filepath.Join(s.backupDir(), "broken-"+time.Now().Format("20060102-150405")+".json")
			os.WriteFile(bad, b, 0o644)
			log.Printf("data.json 을 읽지 못했어요 (%v) → %s 로 보관하고 최근 백업을 찾아요", err, bad)
			if list := s.listBackups(); len(list) > 0 {
				if bb, err := os.ReadFile(filepath.Join(s.backupDir(), list[0].Name)); err == nil && json.Unmarshal(bb, &d) == nil {
					log.Printf("백업 %s 에서 되살렸어요", list[0].Name)
					s.data = &d
				}
			}
		} else {
			s.data = &d
		}
	}
	s.fix()
	s.rev = time.Now().UnixMilli()
	return s
}

func (s *Store) path() string      { return filepath.Join(s.dir, "data.json") }
func (s *Store) backupDir() string { return filepath.Join(s.dir, "backups") }
func (s *Store) filesDir() string  { return filepath.Join(s.dir, "files") }

func (s *Store) fix() {
	d := s.data
	if d.V == 0 {
		d.V = 1
	}
	if d.Sermons == nil {
		d.Sermons = []*Sermon{}
	}
	if d.Series == nil {
		d.Series = []*Series{}
	}
	if d.Illus == nil {
		d.Illus = []*Illustration{}
	}
	if d.Rules == nil {
		d.Rules = []*Rule{}
	}
	if d.Trash == nil {
		d.Trash = []*Sermon{}
	}
}

func nowStamp() string { return time.Now().Format("2006-01-02T15:04:05") }

// save: 반드시 s.mu 를 잡은 채로 불러요
func (s *Store) save() error {
	s.rev = time.Now().UnixMilli()
	b, err := json.MarshalIndent(s.data, "", " ")
	if err != nil {
		return err
	}
	tmp := s.path() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("기록을 저장하지 못했어요: %v", err)
	}
	if err := os.Rename(tmp, s.path()); err != nil {
		return fmt.Errorf("기록을 저장하지 못했어요: %v", err)
	}
	today := time.Now().Format("20060102")
	if s.lastAuto != today {
		s.lastAuto = today
		if !s.hasBackupToday("auto-" + today) {
			s.writeBackup("auto", b)
		}
	}
	return nil
}

func (s *Store) hasBackupToday(prefix string) bool {
	ents, _ := os.ReadDir(s.backupDir())
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), prefix) {
			return true
		}
	}
	return false
}

func (s *Store) writeBackup(kind string, b []byte) (string, error) {
	name := fmt.Sprintf("%s-%s.json", kind, time.Now().Format("20060102-150405"))
	p := filepath.Join(s.backupDir(), name)
	if err := os.WriteFile(p+".tmp", b, 0o644); err != nil {
		return "", err
	}
	os.Rename(p+".tmp", p)
	s.prune(kind, 20)
	return name, nil
}

func (s *Store) prune(kind string, keep int) {
	ents, _ := os.ReadDir(s.backupDir())
	var names []string
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), kind+"-") && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for i, n := range names {
		if i >= keep {
			os.Remove(filepath.Join(s.backupDir(), n))
		}
	}
}

type BackupInfo struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Time    string `json:"time"`
	Size    int64  `json:"size"`
	Sermons int    `json:"sermons"`
}

func (s *Store) listBackups() []BackupInfo {
	ents, _ := os.ReadDir(s.backupDir())
	var out []BackupInfo
	for _, e := range ents {
		n := e.Name()
		if !strings.HasSuffix(n, ".json") || strings.HasPrefix(n, "broken-") {
			continue
		}
		kind, rest, ok := strings.Cut(strings.TrimSuffix(n, ".json"), "-")
		if !ok {
			continue
		}
		if kind == "before" {
			kind = "before-restore"
			rest = strings.TrimPrefix(rest, "restore-")
		}
		t, err := time.ParseInLocation("20060102-150405", rest, time.Local)
		if err != nil {
			continue
		}
		info, _ := e.Info()
		bi := BackupInfo{Name: n, Kind: kind, Time: t.Format("2006-01-02 15:04"), Size: info.Size()}
		out = append(out, bi)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
	return out
}

func (s *Store) backupNow() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(s.data, "", " ")
	if err != nil {
		return "", err
	}
	return s.writeBackup("manual", b)
}

func (s *Store) restore(name string) error {
	if strings.ContainsAny(name, "/\\") || !strings.HasSuffix(name, ".json") {
		return errors.New("백업 이름이 올바르지 않아요")
	}
	b, err := os.ReadFile(filepath.Join(s.backupDir(), name))
	if err != nil {
		return errors.New("백업 파일을 찾지 못했어요")
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return errors.New("백업 파일을 읽지 못했어요")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, _ := json.MarshalIndent(s.data, "", " ")
	if _, err := s.writeBackup("before-restore", cur); err != nil {
		return errors.New("지금 상태를 먼저 보관하지 못해서 멈췄어요")
	}
	s.data = &d
	s.fix()
	return s.save()
}

// ---------------------------------------------------------------- 읽기

// snapshot: 화면에 보낼 전체 기록 (설교 도우미 결과는 무거워서 빼고 '있음' 표시만)
func (s *Store) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	light := make([]map[string]any, 0, len(s.data.Sermons))
	for _, x := range s.data.Sermons {
		light = append(light, sermonLight(x))
	}
	trash := make([]map[string]any, 0, len(s.data.Trash))
	for _, x := range s.data.Trash {
		trash = append(trash, map[string]any{"id": x.ID, "title": x.Title, "date": x.Date, "ref": refsText(x.Passages), "deleted": x.Updated})
	}
	return map[string]any{"rev": s.rev, "sermons": light, "series": s.data.Series, "illus": s.data.Illus, "rules": s.data.Rules, "trash": trash}
}

func refsText(ps []Passage) string {
	var a []string
	for _, p := range ps {
		a = append(a, p.Ref)
	}
	return strings.Join(a, "; ")
}

func sermonLight(x *Sermon) map[string]any {
	b, _ := json.Marshal(x)
	var m map[string]any
	json.Unmarshal(b, &m)
	helper := []string{}
	for k := range x.Helper {
		helper = append(helper, k)
	}
	sort.Strings(helper)
	m["helper"] = helper
	if utf8.RuneCountInString(x.BibleText) > 0 {
		m["hasBibleText"] = true
	}
	return m
}

func (s *Store) sermon(id string) *Sermon {
	for _, x := range s.data.Sermons {
		if x.ID == id {
			return x
		}
	}
	return nil
}

// ---------------------------------------------------------------- HTTP

func (a *App) handleData(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.store.snapshot())
}

func (a *App) handleSermonGet(w http.ResponseWriter, r *http.Request) {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	x := a.store.sermon(r.URL.Query().Get("id"))
	if x == nil {
		jsonError(w, 404, "설교를 찾지 못했어요")
		return
	}
	writeJSON(w, x)
}

var validKinds = map[string]bool{"주일": true, "주일 오후": true, "새벽": true, "수요": true, "금요": true, "성찬": true, "부서": true, "수련회": true, "그 외": true}
var validStatus = map[string]bool{"본문 미정": true, "준비 중": true, "원고 완료": true, "설교함": true}

func (a *App) handleSermonSave(w http.ResponseWriter, r *http.Request) {
	// 화면이 보낸 설교에서 도우미 결과 · 화면용 표시는 빼고 읽어요 (도우미 결과는 따로 저장해요)
	var raw struct {
		Sermon map[string]json.RawMessage `json:"sermon"`
	}
	if err := readJSONBig(r, &raw); err != nil || raw.Sermon == nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	for _, k := range []string{"helper", "hasBibleText", "files"} {
		delete(raw.Sermon, k)
	}
	var x Sermon
	if b, _ := json.Marshal(raw.Sermon); json.Unmarshal(b, &x) != nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	x.Title = strings.TrimSpace(x.Title)
	x.Place = strings.TrimSpace(x.Place)
	if !validKinds[x.Kind] {
		x.Kind = "주일"
	}
	if !validStatus[x.Status] {
		x.Status = "준비 중"
	}
	if _, err := time.Parse("2006-01-02", x.Date); err != nil {
		jsonError(w, 400, "날짜를 확인해 주세요")
		return
	}
	x.Passages = parsePassages(x.PassageText)
	pts := []string{}
	for _, p := range x.Points {
		if strings.TrimSpace(p) != "" {
			pts = append(pts, strings.TrimSpace(p))
		}
	}
	x.Points = pts
	if x.Confessions == nil {
		x.Confessions = []ConfLink{}
	}
	if x.Illus == nil {
		x.Illus = []string{}
	}
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowStamp()
	x.Updated = now
	if x.ID == "" {
		x.ID = "s" + randID(6)
		x.Created = now
		x.Files = []FileRef{}
		s.data.Sermons = append(s.data.Sermons, &x)
	} else {
		old := s.sermon(x.ID)
		if old == nil {
			jsonError(w, 404, "설교를 찾지 못했어요. 지워졌을 수 있어요")
			return
		}
		// 파일 목록과 도우미 결과는 화면이 보낸 것 대신 저장된 것을 지켜요 (따로 다루는 기능이 있어서)
		x.Files = old.Files
		x.Created = old.Created
		x.Helper = old.Helper
		*old = x
	}
	if err := s.save(); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "sermon": s.sermon(x.ID), "rev": s.rev})
}

func (a *App) handleSermonDelete(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	readJSON(r, &in)
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.data.Sermons {
		if x.ID == in.ID {
			s.data.Sermons = append(s.data.Sermons[:i], s.data.Sermons[i+1:]...)
			x.Updated = nowStamp()
			s.data.Trash = append([]*Sermon{x}, s.data.Trash...)
			if len(s.data.Trash) > 50 {
				for _, old := range s.data.Trash[50:] {
					os.RemoveAll(filepath.Join(s.filesDir(), old.ID))
				}
				s.data.Trash = s.data.Trash[:50]
			}
			if err := s.save(); err != nil {
				jsonError(w, 500, err.Error())
				return
			}
			writeJSON(w, map[string]any{"ok": true, "rev": s.rev})
			return
		}
	}
	jsonError(w, 404, "설교를 찾지 못했어요")
}

func (a *App) handleSermonRestore(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	readJSON(r, &in)
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.data.Trash {
		if x.ID == in.ID {
			s.data.Trash = append(s.data.Trash[:i], s.data.Trash[i+1:]...)
			s.data.Sermons = append(s.data.Sermons, x)
			if err := s.save(); err != nil {
				jsonError(w, 500, err.Error())
				return
			}
			writeJSON(w, map[string]any{"ok": true, "rev": s.rev})
			return
		}
	}
	jsonError(w, 404, "휴지통에서 찾지 못했어요")
}

// 설교 도우미 결과를 설교에 붙여요
func (a *App) handleSermonHelper(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string `json:"id"`
		Tab    string `json:"tab"`
		Result any    `json:"result"`
	}
	if err := readJSONBig(r, &in); err != nil || in.Tab == "" {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.sermon(in.ID)
	if x == nil {
		jsonError(w, 404, "설교를 찾지 못했어요")
		return
	}
	if x.Helper == nil {
		x.Helper = map[string]any{}
	}
	if in.Result == nil {
		delete(x.Helper, in.Tab)
	} else {
		x.Helper[in.Tab] = in.Result
	}
	x.Updated = nowStamp()
	if err := s.save(); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "rev": s.rev})
}

// 시리즈 · 예화 · 반복 일정: 같은 꼴(목록에 넣거나 바꾸거나 지우기)
func (a *App) handleSeries(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Series *Series `json:"series"`
		Delete string  `json:"delete"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Delete != "" {
		out := s.data.Series[:0]
		for _, x := range s.data.Series {
			if x.ID != in.Delete {
				out = append(out, x)
			}
		}
		s.data.Series = out
		for _, x := range s.data.Sermons {
			if x.SeriesID == in.Delete {
				x.SeriesID, x.SeriesNo = "", 0
			}
		}
		for _, x := range s.data.Rules {
			if x.SeriesID == in.Delete {
				x.SeriesID = ""
			}
		}
	} else if in.Series != nil {
		x := in.Series
		x.Name = strings.TrimSpace(x.Name)
		if x.Name == "" {
			jsonError(w, 400, "시리즈 이름을 적어 주세요")
			return
		}
		if x.ID == "" {
			x.ID = "r" + randID(5)
			x.Created = nowStamp()
			s.data.Series = append(s.data.Series, x)
		} else {
			found := false
			for i, old := range s.data.Series {
				if old.ID == x.ID {
					x.Created = old.Created
					s.data.Series[i] = x
					found = true
				}
			}
			if !found {
				s.data.Series = append(s.data.Series, x)
			}
		}
	}
	if err := s.save(); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "rev": s.rev, "series": in.Series})
}

func (a *App) handleIllus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Illus  *Illustration `json:"illus"`
		Delete string        `json:"delete"`
	}
	if err := readJSONBig(r, &in); err != nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Delete != "" {
		out := s.data.Illus[:0]
		for _, x := range s.data.Illus {
			if x.ID != in.Delete {
				out = append(out, x)
			}
		}
		s.data.Illus = out
		for _, x := range s.data.Sermons {
			keep := x.Illus[:0]
			for _, id := range x.Illus {
				if id != in.Delete {
					keep = append(keep, id)
				}
			}
			x.Illus = keep
		}
	} else if in.Illus != nil {
		x := in.Illus
		x.Title = strings.TrimSpace(x.Title)
		if x.Title == "" {
			jsonError(w, 400, "예화 제목을 적어 주세요")
			return
		}
		if x.Tags == nil {
			x.Tags = []string{}
		}
		x.Updated = nowStamp()
		if x.ID == "" {
			x.ID = "i" + randID(5)
			x.Created = x.Updated
			s.data.Illus = append(s.data.Illus, x)
		} else {
			found := false
			for i, old := range s.data.Illus {
				if old.ID == x.ID {
					x.Created = old.Created
					s.data.Illus[i] = x
					found = true
				}
			}
			if !found {
				s.data.Illus = append(s.data.Illus, x)
			}
		}
	}
	if err := s.save(); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "rev": s.rev, "illus": in.Illus})
}

func (a *App) handleRule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Rule   *Rule  `json:"rule"`
		Delete string `json:"delete"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Delete != "" {
		out := s.data.Rules[:0]
		for _, x := range s.data.Rules {
			if x.ID != in.Delete {
				out = append(out, x)
			}
		}
		s.data.Rules = out
	} else if in.Rule != nil {
		x := in.Rule
		if !validKinds[x.Kind] {
			x.Kind = "주일"
		}
		if x.Dow < 0 || x.Dow > 6 {
			x.Dow = 0
		}
		if x.Every != 2 {
			x.Every = 1
		}
		if x.ID == "" {
			x.ID = "u" + randID(5)
			s.data.Rules = append(s.data.Rules, x)
		} else {
			found := false
			for i, old := range s.data.Rules {
				if old.ID == x.ID {
					s.data.Rules[i] = x
					found = true
				}
			}
			if !found {
				s.data.Rules = append(s.data.Rules, x)
			}
		}
	}
	if err := s.save(); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "rev": s.rev})
}

// ---------------------------------------------------------------- 첨부 파일

func safeName(n string) string {
	n = filepath.Base(strings.ReplaceAll(n, "\\", "/"))
	n = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/:*?"<>|`, r) {
			return '_'
		}
		return r
	}, n)
	if n == "" || n == "." || n == ".." {
		n = "파일"
	}
	return n
}

func (a *App) handleFileAdd(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)
	mr, err := r.MultipartReader()
	if err != nil {
		jsonError(w, 400, "파일을 받지 못했어요")
		return
	}
	var sid string
	var added []FileRef
	s := a.store
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			jsonError(w, 400, "파일을 받다가 끊겼어요")
			return
		}
		if part.FormName() == "sid" {
			b, _ := io.ReadAll(io.LimitReader(part, 100))
			sid = string(b)
			continue
		}
		if part.FormName() != "file" {
			continue
		}
		s.mu.Lock()
		ok := s.sermon(sid) != nil
		s.mu.Unlock()
		if !ok {
			jsonError(w, 404, "먼저 설교를 저장해 주세요")
			return
		}
		f, err := a.storeFile(sid, part)
		if err != nil {
			jsonError(w, 500, err.Error())
			return
		}
		added = append(added, f)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.sermon(sid)
	if x == nil {
		jsonError(w, 404, "설교를 찾지 못했어요")
		return
	}
	x.Files = append(x.Files, added...)
	x.Updated = nowStamp()
	if err := s.save(); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "files": x.Files, "rev": s.rev})
}

func (a *App) storeFile(sid string, part *multipart.Part) (FileRef, error) {
	name := safeName(part.FileName())
	id := randID(4)
	dir := filepath.Join(a.store.filesDir(), sid)
	os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, id+"_"+name)
	out, err := os.Create(p)
	if err != nil {
		return FileRef{}, errors.New("파일을 저장하지 못했어요")
	}
	n, err := io.Copy(out, part)
	out.Close()
	if err != nil {
		os.Remove(p)
		return FileRef{}, errors.New("파일을 받다가 끊겼어요")
	}
	return FileRef{ID: id, Name: name, Size: n, Added: nowStamp()}, nil
}

func (a *App) filePath(sid, fid string) (string, *Sermon, int) {
	s := a.store
	x := s.sermon(sid)
	if x == nil {
		return "", nil, -1
	}
	for i, f := range x.Files {
		if f.ID == fid {
			return filepath.Join(s.filesDir(), sid, f.ID+"_"+f.Name), x, i
		}
	}
	return "", x, -1
}

func (a *App) handleFileAction(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SID    string `json:"sid"`
		FID    string `json:"fid"`
		Action string `json:"action"` // open | reveal | remove
	}
	readJSON(r, &in)
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	p, x, i := a.filePath(in.SID, in.FID)
	if i < 0 {
		jsonError(w, 404, "파일을 찾지 못했어요")
		return
	}
	switch in.Action {
	case "open":
		openPaths([]string{p})
	case "reveal":
		revealPaths([]string{p})
	case "remove":
		os.Remove(p)
		x.Files = append(x.Files[:i], x.Files[i+1:]...)
		x.Updated = nowStamp()
		if err := s.save(); err != nil {
			jsonError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true, "files": x.Files, "rev": s.rev})
}

func (a *App) handleBackups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"backups": a.store.listBackups(), "dir": a.store.backupDir()})
}

func (a *App) handleBackupNow(w http.ResponseWriter, r *http.Request) {
	name, err := a.store.backupNow()
	if err != nil {
		jsonError(w, 500, "백업하지 못했어요: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "name": name, "backups": a.store.listBackups()})
}

func (a *App) handleBackupRestore(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	readJSON(r, &in)
	if err := a.store.restore(in.Name); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (a *App) handleBackupReveal(w http.ResponseWriter, r *http.Request) {
	openPaths([]string{a.store.backupDir()})
	writeJSON(w, map[string]any{"ok": true})
}

func readJSONBig(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(v)
}
