// SermonROAD — 개혁주의 설교자를 위한 설교 기록 · 준비 앱 (macOS, Apple Silicon)
// made by. Nevertheless_D
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// build is the app version, set with -ldflags "-X main.build=1.2.3".
var build = "0.0.0-dev"

//go:embed web
var webFiles embed.FS

type Settings struct {
	Appearance  string         `json:"appearance"` // system | light | dark
	AutoUpdate  bool           `json:"autoUpdate"`
	ClaudeMode  string         `json:"claudeMode"` // api | copy
	ShowBoth    bool           `json:"showBoth"`   // 도우미 화면에 다른 방법 버튼도
	Model       string         `json:"model"`
	Standards   []string       `json:"standards"` // 기준 문서 (wcf wlc wsc hc bc cd)
	ExtraPrompt string         `json:"extraPrompt"`
	Library     string         `json:"library"` // 서재 폴더
	UI          map[string]any `json:"ui"`
}

type App struct {
	support string
	token   string
	port    int

	mu       sync.Mutex
	settings Settings

	store *Store
	lib   *Library
	upd   *Updater

	lastSeen atomic.Int64
}

var app *App

// started 는 단조 시계 기준점이에요 (맥이 잠자는 동안 늘어나지 않아요)
var started = time.Now()

func touch() { app.lastSeen.Store(int64(time.Since(started))) }

const idleLimit = 90 * time.Minute

func env(k string) string { return os.Getenv("SERMONROAD_" + k) }

// defaultPort: 늘 같은 주소를 쓰면 엔진이 다시 켜져도 열려 있던 창이 이어지고, 진단 페이지도 찾기 쉬워요.
const defaultPort = 47931

func main() {
	log.SetFlags(log.LstdFlags)
	home, _ := os.UserHomeDir()
	support := env("SUPPORT")
	if support == "" {
		switch runtime.GOOS {
		case "darwin":
			support = filepath.Join(home, "Library", "Application Support", "SermonROAD")
		default:
			support = filepath.Join(home, ".sermonroad")
		}
	}
	if env("SUPERVISOR") == "1" {
		os.MkdirAll(support, 0o755)
		runSupervisor(support)
		return
	}
	token := env("TOKEN")
	if token == "" {
		token = randID(16)
	}
	app = &App{support: support, token: token}
	os.MkdirAll(support, 0o755)
	os.Rename(filepath.Join(support, "engine.log"), filepath.Join(support, "engine.prev.log"))
	if f, err := os.OpenFile(filepath.Join(support, "engine.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	}
	app.loadSettings()
	touch()
	app.store = NewStore(support)
	app.lib = NewLibrary(support)
	app.upd = NewUpdater(support)
	if s := app.Settings(); s.Library != "" {
		go func() { defer recoverLog("library scan"); app.lib.SetFolder(s.Library) }()
	}

	go heartbeat(support)
	ln := listen(env("PORT"), env("PORT_WAIT") == "1")
	app.port = ln.Addr().(*net.TCPAddr).Port
	writeFileAtomic(filepath.Join(support, "token"), []byte(token), 0o600)
	writeFileAtomic(filepath.Join(support, "port"), []byte(strconv.Itoa(app.port)), 0o644)
	writeFileAtomic(filepath.Join(support, "lastport"), []byte(strconv.Itoa(app.port)), 0o644)
	log.Printf("SermonROAD %s listening on %d (pid %d, 지킴이 %v)", build, app.port, os.Getpid(), env("CHILD") == "1")

	if app.port != defaultPort {
		go func() {
			defer recoverLog("diag listener")
			if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", defaultPort)); err == nil {
				m := http.NewServeMux()
				m.HandleFunc("/diag", app.handleDiag)
				http.Serve(ln, hostGuard(m))
			}
		}()
	}
	go app.idleWatcher()
	go app.watchParent()
	go func() { defer recoverLog("autoLoop"); app.upd.autoLoop() }()
	srv := &http.Server{Handler: hostGuard(recordRequests(app.routes())), ErrorLog: log.Default()}
	log.Fatal(srv.Serve(ln))
}

func listen(pref string, wait bool) net.Listener {
	p, err := strconv.Atoi(pref)
	if err != nil || p <= 0 {
		p = defaultPort
	}
	tries := 1
	if wait {
		tries = 20
	}
	for i := 0; i < tries; i++ {
		if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p)); err == nil {
			return ln
		}
		time.Sleep(150 * time.Millisecond)
	}
	if p != defaultPort {
		if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", defaultPort)); err == nil {
			return ln
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	return ln
}

// ---------------------------------------------------------------- lifecycle

func (a *App) idleWatcher() {
	defer recoverLog("idleWatcher")
	// Dock 앱이 켠 엔진은 앱이 켜져 있는 동안 계속 켜 둬요. 앱을 종료하면 엔진도 함께 꺼져요.
	if nativeShell() {
		return
	}
	for range time.Tick(15 * time.Second) {
		idle := time.Since(started) - time.Duration(a.lastSeen.Load())
		if idle > idleLimit && !a.upd.Busy() {
			log.Printf("idle for %s, shutting down", idle.Round(time.Second))
			a.shutdownWhy("오래 쉼")
		}
	}
}

func nativeShell() bool { return env("SHELL") == "native" }

// watchParent: 지킴이(부모)가 사라지면 엔진도 끝나요 — 혼자 남아 몰래 돌지 않게
func (a *App) watchParent() {
	if env("CHILD") != "1" {
		return
	}
	ppid := os.Getppid()
	for range time.Tick(2 * time.Second) {
		if os.Getppid() != ppid {
			a.shutdownWhy("지킴이가 사라짐")
		}
	}
}

func (a *App) shutdownWhy(why string) {
	log.Printf("shutdown: %s", why)
	markCleanExit(a.support, why)
	if b, err := os.ReadFile(filepath.Join(a.support, "port")); err == nil && string(b) == strconv.Itoa(a.port) {
		os.Remove(filepath.Join(a.support, "port"))
	}
	os.Exit(0)
}

// ---------------------------------------------------------------- settings

func (a *App) settingsPath() string { return filepath.Join(a.support, "settings.json") }

func (a *App) loadSettings() {
	a.settings = Settings{Appearance: "system", AutoUpdate: true, ClaudeMode: "api", ShowBoth: true, Model: helperModels[0]["id"],
		Standards:   []string{"wcf", "wlc", "wsc", "hc", "cd"},
		ExtraPrompt: "성경으로 성경을 해석하고, 그리스도 중심으로 보며, 언약의 흐름 안에서 본문을 읽어 주세요. 자료에 없는 내용은 반드시 따로 표시해 주세요.",
		UI:          map[string]any{}}
	if b, err := os.ReadFile(a.settingsPath()); err == nil {
		_ = json.Unmarshal(b, &a.settings)
	}
	if a.settings.UI == nil {
		a.settings.UI = map[string]any{}
	}
	if a.settings.ClaudeMode != "copy" {
		a.settings.ClaudeMode = "api"
	}
}

func (a *App) saveSettings() {
	b, _ := json.MarshalIndent(a.settings, "", "  ")
	writeFileAtomic(a.settingsPath(), b, 0o644)
}

func (a *App) Settings() Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.settings
	s.UI = make(map[string]any, len(a.settings.UI))
	for k, v := range a.settings.UI {
		s.UI[k] = v
	}
	s.Standards = append([]string(nil), a.settings.Standards...)
	return s
}

func (a *App) settingsJSON() map[string]any {
	s := a.Settings()
	home, _ := os.UserHomeDir()
	return map[string]any{
		"appearance":  s.Appearance,
		"autoUpdate":  s.AutoUpdate,
		"claudeMode":  s.ClaudeMode,
		"showBoth":    s.ShowBoth,
		"model":       s.Model,
		"standards":   s.Standards,
		"extraPrompt": s.ExtraPrompt,
		"library":     s.Library,
		"ui":          s.UI,
		"home":        home,
		"build":       build,
		"platform":    runtime.GOOS + "/" + runtime.GOARCH,
		"support":     a.support,
		"exportDir":   exportDir(),
	}
}

func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var in map[string]json.RawMessage
		if err := readJSON(r, &in); err != nil {
			jsonError(w, 400, "bad request")
			return
		}
		a.mu.Lock()
		for k, v := range in {
			switch k {
			case "appearance":
				json.Unmarshal(v, &a.settings.Appearance)
			case "autoUpdate":
				json.Unmarshal(v, &a.settings.AutoUpdate)
			case "claudeMode":
				json.Unmarshal(v, &a.settings.ClaudeMode)
				if a.settings.ClaudeMode != "copy" {
					a.settings.ClaudeMode = "api"
				}
			case "showBoth":
				json.Unmarshal(v, &a.settings.ShowBoth)
			case "model":
				json.Unmarshal(v, &a.settings.Model)
			case "standards":
				json.Unmarshal(v, &a.settings.Standards)
			case "extraPrompt":
				json.Unmarshal(v, &a.settings.ExtraPrompt)
			case "ui":
				var ui map[string]any
				if json.Unmarshal(v, &ui) == nil {
					for kk, vv := range ui {
						a.settings.UI[kk] = vv
					}
				}
			}
		}
		a.saveSettings()
		a.mu.Unlock()
	}
	writeJSON(w, a.settingsJSON())
}

// ---------------------------------------------------------------- routes

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFiles, "web")
	static := http.FileServer(http.FS(sub))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			if r.URL.Query().Get("t") != a.token {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				io.WriteString(w, "<meta charset=utf-8><body style='font:15px -apple-system;padding:40px'>SermonROAD 앱을 다시 실행해 주세요.</body>")
				return
			}
			b, _ := webFiles.ReadFile("web/index.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(b)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".js") || strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Cache-Control", "no-cache")
		} else if strings.HasPrefix(r.URL.Path, "/fonts/") {
			w.Header().Set("Cache-Control", "public, max-age=604800")
		}
		static.ServeHTTP(w, r)
	})

	mux.HandleFunc("/diag", a.handleDiag)
	mux.HandleFunc("/print", a.handlePrint)

	api := func(path string, h http.HandlerFunc) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			t := r.Header.Get("x-token")
			if t == "" {
				t = r.URL.Query().Get("t")
			}
			if t != a.token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			touch()
			defer func() {
				if v := recover(); v != nil {
					log.Printf("PANIC in %s: %v\n%s", path, v, debug.Stack())
					jsonError(w, 500, "앱 안에서 문제가 생겼어요. 다시 시도해 주세요 (진단 정보에 남았어요)")
				}
			}()
			h(w, r)
		})
	}

	api("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "build": build, "shell": env("SHELL")})
	})
	api("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); a.shutdownWhy("종료 버튼") }()
	})
	api("/api/clientlog", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Items []string `json:"items"`
		}
		readJSON(r, &in)
		for i, x := range in.Items {
			if i >= 30 {
				break
			}
			if len(x) > 600 {
				x = x[:600]
			}
			diag.clientErr(x)
			log.Printf("client: %s", x)
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/poll", func(w http.ResponseWriter, r *http.Request) {
		a.store.mu.Lock()
		rev := a.store.rev
		a.store.mu.Unlock()
		a.lib.mu.Lock()
		lrev, busy := a.lib.rev, a.lib.busy
		a.lib.mu.Unlock()
		writeJSON(w, map[string]any{"build": build, "rev": rev, "libRev": lrev, "libBusy": busy, "update": a.upd.State(), "shell": env("SHELL")})
	})
	api("/api/crash", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "report": crashReport(a.support)})
	})
	api("/api/settings", a.handleSettings)
	api("/api/bible", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"books": books})
	})
	api("/api/parse", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Text string `json:"text"`
		}
		readJSON(r, &in)
		ps := parsePassages(in.Text)
		if ps == nil {
			ps = []Passage{}
		}
		writeJSON(w, map[string]any{"passages": ps})
	})

	// 설교 기록
	api("/api/data", a.handleData)
	api("/api/sermon", a.handleSermonGet)
	api("/api/sermon/save", a.handleSermonSave)
	api("/api/sermon/delete", a.handleSermonDelete)
	api("/api/sermon/restore", a.handleSermonRestore)
	api("/api/sermon/helper", a.handleSermonHelper)
	api("/api/series/save", a.handleSeries)
	api("/api/illus/save", a.handleIllus)
	api("/api/rule/save", a.handleRule)
	api("/api/file/add", a.handleFileAdd)
	api("/api/file/action", a.handleFileAction)
	api("/api/backups", a.handleBackups)
	api("/api/backup/now", a.handleBackupNow)
	api("/api/backup/restore", a.handleBackupRestore)
	api("/api/backup/reveal", a.handleBackupReveal)
	api("/api/export", a.handleExport)
	api("/api/import/parse", a.handleImportParse)
	api("/api/import/commit", a.handleImportCommit)
	api("/api/import/undo", a.handleImportUndo)
	api("/api/import/template", a.handleImportTemplate)

	// 서재 · 설교 도우미
	api("/api/library", a.handleLibrary)
	api("/api/library/folder", a.handleLibraryFolder)
	api("/api/library/rescan", a.handleLibraryRescan)
	api("/api/library/book", a.handleLibraryBook)
	api("/api/library/find", a.handleLibraryFind)
	api("/api/library/page", a.handleLibraryPage)
	api("/api/library/open", a.handleLibraryOpen)
	api("/api/helper/run", a.handleHelperRun)
	api("/api/helper/job", a.handleHelperJob)
	api("/api/helper/prompt", a.handleHelperPrompt)
	api("/api/claude", a.handleClaudeState)
	api("/api/claude/key", a.handleClaudeKey)
	api("/api/claude/test", a.handleClaudeTest)

	// 업데이트
	api("/api/update/check", func(w http.ResponseWriter, r *http.Request) {
		a.upd.Check(true)
		writeJSON(w, a.upd.State())
	})
	api("/api/update/install", func(w http.ResponseWriter, r *http.Request) {
		go func() { defer recoverLog("install"); a.upd.Install() }()
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/update/rollback", func(w http.ResponseWriter, r *http.Request) {
		if err := a.upd.Rollback(); err != nil {
			jsonError(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/update/restart", func(w http.ResponseWriter, r *http.Request) {
		if err := a.upd.Restart(a.port, a.token); err != nil {
			jsonError(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/open-url", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			URL string `json:"url"`
		}
		readJSON(r, &in)
		if strings.HasPrefix(in.URL, "https://") {
			openPaths([]string{in.URL})
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	return mux
}

// Only answer requests addressed to 127.0.0.1 / localhost (DNS rebinding protection).
func hostGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------- utils

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func randID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeFileAtomic(path string, b []byte, perm os.FileMode) {
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, perm) == nil {
		os.Rename(tmp, path)
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func dirExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// composeHangul: 맥 파일 이름 · PDF 글자의 풀어 쓴 한글(NFD)을 모아 쓴 글자로
func composeHangul(s string) string {
	rs := []rune(s)
	out := make([]rune, 0, len(rs))
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r >= 0x1100 && r <= 0x1112 && i+1 < len(rs) && rs[i+1] >= 0x1161 && rs[i+1] <= 0x1175 {
			l, v := r-0x1100, rs[i+1]-0x1161
			t := rune(0)
			i++
			if i+1 < len(rs) && rs[i+1] >= 0x11A8 && rs[i+1] <= 0x11C2 {
				t = rs[i+1] - 0x11A7
				i++
			}
			out = append(out, 0xAC00+(l*21+v)*28+t)
			continue
		}
		if r >= 0xAC00 && r <= 0xD7A3 && (r-0xAC00)%28 == 0 && i+1 < len(rs) && rs[i+1] >= 0x11A8 && rs[i+1] <= 0x11C2 {
			out = append(out, r+rs[i+1]-0x11A7)
			i++
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func init() {
	mime.AddExtensionType(".js", "text/javascript")
	mime.AddExtensionType(".woff2", "font/woff2")
}

func recoverLog(where string) {
	if v := recover(); v != nil {
		log.Printf("PANIC in %s: %v\n%s", where, v, debug.Stack())
	}
}

// crashReport: 지난번 엔진이 갑자기 멈췄다면 그 기록을 돌려줘요
func crashReport(support string) string {
	var parts []string
	if b, err := os.ReadFile(filepath.Join(support, "engine.prev.out")); err == nil {
		t := string(b)
		if strings.Contains(t, "panic") || strings.Contains(t, "fatal error") || strings.Contains(t, "PANIC") {
			if len(t) > 6000 {
				t = t[len(t)-6000:]
			}
			parts = append(parts, "[지난번 실행]\n"+t)
		}
	}
	if b, err := os.ReadFile(filepath.Join(support, "engine.log")); err == nil {
		t := string(b)
		if i := strings.Index(t, "PANIC"); i >= 0 {
			t = t[i:]
			if len(t) > 6000 {
				t = t[:6000]
			}
			parts = append(parts, "[이번 실행]\n"+t)
		}
	}
	return strings.Join(parts, "\n\n")
}
