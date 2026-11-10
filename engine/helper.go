package main

// 설교 도우미 — 서재에서 고른 자료를 근거로 Claude 에게 정리를 부탁해요.
//   연결 방법(설정에서 고름)
//     api  : 앱이 Claude(Anthropic API)를 바로 불러요. API 키는 맥의 키체인에만 저장해요.
//     copy : 지금 쓰는 Claude 구독으로 — 앱이 요청문과 자료를 만들어 복사해 주면, Claude 에 붙여 넣어요.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var helperTabs = map[string]struct{ label, task string }{
	"sum":     {"정리 · 요약", "자료들이 이 본문에 대해 말하는 핵심을 4~7개 항목으로 정리하고 요약해 주세요. 자료들 사이에 해석이 갈리는 곳이 있으면 그것도 한 항목으로 밝혀 주세요. 각 항목은 head(짧은 제목)와 body(2~4문장)로 써 주세요."},
	"str":     {"구조", "본문의 구조를 정리해 주세요. 문맥(앞뒤 흐름), 문단 나눔, 핵심 단어와 반복 표현(원어가 중요하면 음역과 뜻), 문법 · 시제에서 눈여겨볼 점, 본문의 중심 주제를 4~7개 항목으로. head 에는 '문맥', '문단', '핵심 단어'처럼 짧게 쓰세요."},
	"med":     {"묵상", "목사님이 이 본문을 천천히 묵상할 수 있도록 질문 5~7개를 만들어 주세요. 삼위 하나님에 대해, 그리스도와 복음에 대해, 성도와 교회의 삶에 대해 묻는 질문을 고루 담아 주세요. head 는 비워 두고 body 에 질문 한 문장을 쓰세요."},
	"outline": {"설교 개요", "설교 개요를 만들어 주세요. thesis 에 중심 명제를 한 문장으로 쓰고, items 는 차례대로 서론, 대지들(Ⅰ. Ⅱ. Ⅲ. — 각 대지 아래 소지는 body 안에 '· '로 시작하는 줄로), 적용, 결론이에요. head 는 '서론 — ', 'Ⅰ. 대지 제목 (28절) — ', '적용 — ', '결론 — ' 처럼 쓰세요."},
	"ref":     {"개혁주의 관점", "개혁주의 관점에서 이 본문을 점검해 주세요. 관련 표준문서 조항(문서 이름과 정확한 장 · 항 · 문 · 주일 번호), 조심해야 할 해석과 그 까닭, 그리스도 중심으로 읽는 길, 함께 볼 성경 구절(성경으로 성경을 해석)을 4~7개 항목으로. head 에는 '관련 조항', '조심할 해석', '그리스도 중심', '성경으로 성경을'처럼 쓰세요."},
}

var helperModels = []map[string]string{
	{"id": "claude-sonnet-5-5", "label": "Sonnet 5.5 — 추천 (빠르고 깊이 있음)"},
	{"id": "claude-opus-5-5", "label": "Opus 5.5 — 가장 깊이 (느리고 요금이 더 들어요)"},
	{"id": "claude-haiku-5-5", "label": "Haiku 5.5 — 가장 빠르고 저렴함"},
}

var standardNames = map[string]string{
	"wcf": "웨스트민스터 신앙고백", "wlc": "웨스트민스터 대요리문답", "wsc": "웨스트민스터 소요리문답",
	"hc": "하이델베르크 요리문답", "bc": "벨직 신앙고백", "cd": "도르트 신조",
}

type HelperSource struct {
	Doc  string `json:"doc"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

type HelperReq struct {
	Ref       string         `json:"ref"`
	Tab       string         `json:"tab"`
	BibleText string         `json:"bibleText"`
	Sources   []HelperSource `json:"sources"`
	Note      string         `json:"note"` // 이번에만 덧붙일 말
}

type builtPrompt struct {
	System  string
	User    string
	Sources []map[string]any // S1 → 자료 이름 · 쪽
}

func (a *App) buildPrompt(in HelperReq, copyMode bool) (*builtPrompt, error) {
	tab, ok := helperTabs[in.Tab]
	if !ok {
		return nil, errors.New("알 수 없는 도우미 종류예요")
	}
	ps := parsePassages(in.Ref)
	ref := refsText(ps)
	if strings.TrimSpace(ref) == "" {
		return nil, errors.New("본문을 먼저 적어 주세요")
	}
	s := a.Settings()
	var stds []string
	for _, k := range s.Standards {
		if n := standardNames[k]; n != "" {
			stds = append(stds, n)
		}
	}
	if len(stds) == 0 {
		stds = []string{"웨스트민스터 표준문서", "하이델베르크 요리문답", "도르트 신조"}
	}
	var sys strings.Builder
	sys.WriteString("당신은 개혁주의 설교자를 돕는 설교 준비 도우미입니다. 목사님이 직접 고른 참고 자료(주석 · 강해 · 신학서 PDF 에서 뽑은 쪽들)를 근거로 답합니다.\n\n원칙\n")
	sys.WriteString("1. 성경은 하나님의 무오한 말씀이고, 해석의 최종 기준은 성경 자신입니다 (성경으로 성경을 해석).\n")
	sys.WriteString("2. 기준 문서: " + strings.Join(stds, ", ") + ". 이 문서들과 어긋나는 해석을 하지 않고, 관련 조항이 있으면 밝힙니다.\n")
	sys.WriteString("3. 그리스도 중심으로, 구속사와 언약의 흐름 안에서 본문을 읽습니다.\n")
	if copyMode {
		sys.WriteString("4. 될 수 있는 대로 제공된 자료에 근거하고, 각 항목 끝에 근거가 된 자료 번호와 쪽을 [S1 313쪽] 처럼 적습니다. 자료에 없는 내용을 덧붙일 때는 항목 끝에 (자료 밖) 이라고 적습니다. 쪽 번호를 지어내지 않습니다.\n")
	} else {
		sys.WriteString("4. 될 수 있는 대로 제공된 자료에 근거하고, 각 항목의 sources 에 근거가 된 자료 번호(S1 …)와 쪽을 적습니다. 자료에 없는 내용을 덧붙이는 항목은 outside 를 true 로 합니다. 쪽 번호를 지어내지 않습니다.\n")
	}
	sys.WriteString("5. 한국어로, 설교자가 바로 쓸 수 있게 간결하고 분명하게 씁니다. 자료 문장을 길게 그대로 옮기지 말고 요약합니다.\n")
	if t := strings.TrimSpace(s.ExtraPrompt); t != "" {
		sys.WriteString("\n목사님이 늘 덧붙이는 말: " + t + "\n")
	}

	var u strings.Builder
	fmt.Fprintf(&u, "본문: %s\n", ref)
	if t := strings.TrimSpace(in.BibleText); t != "" {
		if len(t) > 12000 {
			t = t[:12000]
		}
		fmt.Fprintf(&u, "\n목사님이 넣은 성경 본문:\n%s\n", t)
	}
	n := len(in.Sources)
	var srcs []map[string]any
	if n > 0 {
		budget := 90000
		per := budget / n
		if per < 6000 {
			per = 6000
		}
		u.WriteString("\n참고 자료 (목사님의 서재에서 고른 쪽들):\n")
		for i, src := range in.Sources {
			name, text, err := a.lib.PagesText(src.Doc, src.From, src.To, per)
			if err != nil {
				return nil, err
			}
			id := fmt.Sprintf("S%d", i+1)
			fmt.Fprintf(&u, "\n<자료 id=\"%s\" 제목=\"%s\" 쪽=\"%d-%d\">\n%s</자료>\n", id, name, src.From, src.To, text)
			srcs = append(srcs, map[string]any{"id": id, "doc": src.Doc, "name": name, "from": src.From, "to": src.To})
		}
	} else {
		u.WriteString("\n(이번에는 참고 자료가 없어요. 자료 없이 답하는 모든 항목은 자료 밖 내용이에요.)\n")
	}
	if t := strings.TrimSpace(in.Note); t != "" {
		fmt.Fprintf(&u, "\n이번에 덧붙이는 말: %s\n", t)
	}
	fmt.Fprintf(&u, "\n할 일 — %s: %s\n", tab.label, tab.task)
	if copyMode {
		u.WriteString("\n답은 마크다운으로 써 주세요. 중심 명제가 있으면 맨 위에 '**중심 명제** — …' 한 줄로, 항목은 번호 목록으로 써 주세요.\n")
	}
	return &builtPrompt{System: sys.String(), User: u.String(), Sources: srcs}, nil
}

// ---------------------------------------------------------------- 작업 (Claude 부르기)

type helperJob struct {
	ID      string         `json:"id"`
	Status  string         `json:"status"` // running | done | error
	Tab     string         `json:"tab"`
	Ref     string         `json:"ref"`
	Result  map[string]any `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
	Started int64          `json:"started"`
}

var jobs = struct {
	sync.Mutex
	m map[string]*helperJob
}{m: map[string]*helperJob{}}

func (a *App) handleHelperRun(w http.ResponseWriter, r *http.Request) {
	var in HelperReq
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	bp, err := a.buildPrompt(in, false)
	if err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	key := claudeKey(a.support)
	if key == "" {
		jsonError(w, 400, "Claude API 키가 아직 없어요. 설정 › Claude 연결에서 키를 넣어 주세요")
		return
	}
	j := &helperJob{ID: randID(6), Status: "running", Tab: in.Tab, Ref: refsText(parsePassages(in.Ref)), Started: time.Now().UnixMilli()}
	jobs.Lock()
	jobs.m[j.ID] = j
	jobs.Unlock()
	model := a.Settings().Model
	go func() {
		defer recoverLog("helper job")
		res, err := callClaude(a.support, key, model, bp)
		jobs.Lock()
		defer jobs.Unlock()
		if err != nil {
			j.Status, j.Error = "error", err.Error()
			log.Printf("도우미 실패 (%s %s): %v", in.Tab, j.Ref, err)
			return
		}
		res["sources"] = bp.Sources
		res["tab"] = in.Tab
		res["ref"] = j.Ref
		res["mode"] = "api"
		res["model"] = model
		res["at"] = nowStamp()
		j.Status, j.Result = "done", res
	}()
	writeJSON(w, j)
}

func (a *App) handleHelperJob(w http.ResponseWriter, r *http.Request) {
	jobs.Lock()
	defer jobs.Unlock()
	j := jobs.m[r.URL.Query().Get("id")]
	if j == nil {
		jsonError(w, 404, "작업을 찾지 못했어요")
		return
	}
	writeJSON(w, j)
}

// 구독 방식: 붙여 넣을 요청문 만들기
func (a *App) handleHelperPrompt(w http.ResponseWriter, r *http.Request) {
	var in HelperReq
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "보낸 내용을 읽지 못했어요")
		return
	}
	bp, err := a.buildPrompt(in, true)
	if err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	text := bp.System + "\n---\n\n" + bp.User
	writeJSON(w, map[string]any{"text": text, "chars": len([]rune(text)), "sources": bp.Sources, "ref": refsText(parsePassages(in.Ref))})
}

var resultTool = map[string]any{
	"name":        "sermon_helper_result",
	"description": "설교 도우미의 정리 결과를 돌려줍니다.",
	"input_schema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"thesis": map[string]any{"type": "string", "description": "중심 명제 (설교 개요일 때만, 한 문장)"},
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"head": map[string]any{"type": "string"},
						"body": map[string]any{"type": "string"},
						"sources": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":       "object",
								"properties": map[string]any{"id": map[string]any{"type": "string"}, "page": map[string]any{"type": "integer"}},
								"required":   []string{"id"},
							},
						},
						"outside": map[string]any{"type": "boolean", "description": "제공된 자료에 없는 내용이면 true"},
					},
					"required": []string{"body", "outside"},
				},
			},
			"caution": map[string]any{"type": "string", "description": "자료가 부족하거나 조심할 점이 있으면 한두 문장"},
		},
		"required": []string{"items"},
	},
}

func apiURL() string {
	if u := env("CLAUDE_URL"); u != "" {
		return u
	}
	return "https://api.anthropic.com/v1/messages"
}

func callClaude(support, key, model string, bp *builtPrompt) (map[string]any, error) {
	if model == "" {
		model = helperModels[0]["id"]
	}
	body := map[string]any{
		"model":       model,
		"max_tokens":  8000,
		"system":      bp.System,
		"messages":    []map[string]any{{"role": "user", "content": bp.User}},
		"tools":       []any{resultTool},
		"tool_choice": map[string]any{"type": "tool", "name": "sermon_helper_result"},
	}
	var resp struct {
		Content []struct {
			Type  string         `json:"type"`
			Input map[string]any `json:"input"`
			Text  string         `json:"text"`
		} `json:"content"`
		Usage struct {
			In  int `json:"input_tokens"`
			Out int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}
	if err := claudePost(key, body, 6*time.Minute, &resp); err != nil {
		return nil, err
	}
	addUsage(support, resp.Usage.In, resp.Usage.Out)
	for _, c := range resp.Content {
		if c.Type == "tool_use" && c.Input != nil {
			if resp.StopReason == "max_tokens" {
				c.Input["caution"] = strings.TrimSpace(fmt.Sprint(c.Input["caution"], " 답이 길어서 끝부분이 잘렸을 수 있어요."))
			}
			return c.Input, nil
		}
	}
	return nil, errors.New("Claude 의 답을 읽지 못했어요. 다시 시도해 주세요")
}

func claudePost(key string, body map[string]any, timeout time.Duration, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", apiURL(), bytes.NewReader(b))
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "SermonROAD/"+build)
	res, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "Timeout") || strings.Contains(err.Error(), "deadline") {
			return errors.New("Claude 의 답이 너무 오래 걸려요. 자료를 조금 줄이거나 잠시 뒤 다시 시도해 주세요")
		}
		return errors.New("Claude 에 연결하지 못했어요. 인터넷 연결을 확인해 주세요")
	}
	defer res.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != 200 {
		var e struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.Unmarshal(rb, &e)
		msg := e.Error.Message
		switch {
		case res.StatusCode == 401:
			return errors.New("API 키가 맞지 않아요. 설정 › Claude 연결에서 키를 다시 넣어 주세요")
		case res.StatusCode == 403:
			return errors.New("이 API 키로는 쓸 수 없어요 (권한 없음): " + msg)
		case res.StatusCode == 429:
			return errors.New("잠깐 사이에 요청이 많았어요. 1분쯤 뒤 다시 시도해 주세요")
		case res.StatusCode == 529 || res.StatusCode >= 500:
			return errors.New("Claude 가 지금 바빠요. 잠시 뒤 다시 시도해 주세요")
		case strings.Contains(strings.ToLower(msg), "credit"):
			return errors.New("API 사용 금액(크레딧)이 모자라요. Anthropic 콘솔에서 충전해 주세요")
		case strings.Contains(strings.ToLower(msg), "too long") || strings.Contains(strings.ToLower(msg), "prompt is too long"):
			return errors.New("보낼 자료가 너무 많아요. 자료를 몇 개 빼고 다시 시도해 주세요")
		case strings.Contains(strings.ToLower(msg), "model"):
			return errors.New("고른 모델을 쓸 수 없어요. 설정에서 다른 모델을 골라 주세요: " + msg)
		}
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", res.StatusCode)
		}
		return errors.New("Claude 가 요청을 받지 않았어요: " + msg)
	}
	if err := json.Unmarshal(rb, out); err != nil {
		return errors.New("Claude 의 답을 읽지 못했어요")
	}
	return nil
}

// ---------------------------------------------------------------- 사용량 (이번 달)

var usageMu sync.Mutex

func addUsage(support string, in, out int) {
	usageMu.Lock()
	defer usageMu.Unlock()
	p := filepath.Join(support, "usage.json")
	m := map[string]map[string]int{}
	if b, err := os.ReadFile(p); err == nil {
		json.Unmarshal(b, &m)
	}
	k := time.Now().Format("2006-01")
	if m[k] == nil {
		m[k] = map[string]int{}
	}
	m[k]["calls"]++
	m[k]["in"] += in
	m[k]["out"] += out
	b, _ := json.Marshal(m)
	writeFileAtomic(p, b, 0o644)
}

func monthUsage(support string) map[string]int {
	usageMu.Lock()
	defer usageMu.Unlock()
	m := map[string]map[string]int{}
	if b, err := os.ReadFile(filepath.Join(support, "usage.json")); err == nil {
		json.Unmarshal(b, &m)
	}
	if u := m[time.Now().Format("2006-01")]; u != nil {
		return u
	}
	return map[string]int{"calls": 0, "in": 0, "out": 0}
}

// ---------------------------------------------------------------- API 키 (맥 키체인 · Windows 자격 증명 관리자)

const keychainService = "SermonROAD Claude API"

var reKey = regexp.MustCompile(`^[A-Za-z0-9_\-]{20,300}$`)

func claudeKey(support string) string {
	if k := env("CLAUDE_KEY"); k != "" {
		return k
	}
	if isWindows {
		return winCredGet(keychainService)
	}
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("/usr/bin/security", "find-generic-password", "-a", "SermonROAD", "-s", keychainService, "-w").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	b, _ := os.ReadFile(filepath.Join(support, "claude-key"))
	return strings.TrimSpace(string(b))
}

func setClaudeKey(support, key string) error {
	if key == "" {
		if isWindows {
			winCredDelete(keychainService)
		} else if runtime.GOOS == "darwin" {
			exec.Command("/usr/bin/security", "delete-generic-password", "-a", "SermonROAD", "-s", keychainService).Run()
		} else {
			os.Remove(filepath.Join(support, "claude-key"))
		}
		return nil
	}
	if !reKey.MatchString(key) {
		return errors.New("API 키 모양이 아니에요. sk-ant- 로 시작하는 키를 그대로 붙여 넣어 주세요")
	}
	if isWindows {
		// Windows 자격 증명 관리자에 저장해요 (맥의 키체인과 같은 곳)
		if err := winCredSet(keychainService, key); err != nil {
			return fmt.Errorf("Windows 자격 증명 관리자에 저장하지 못했어요: %v", err)
		}
		if claudeKey(support) != key {
			return errors.New("자격 증명 관리자에 저장했지만 다시 읽지 못했어요")
		}
		return nil
	}
	if runtime.GOOS == "darwin" {
		// 키를 명령줄에 드러내지 않도록 security 의 입력으로 넘겨요
		cmd := exec.Command("/usr/bin/security", "-i")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("add-generic-password -U -a SermonROAD -s \"%s\" -l \"%s\" -w \"%s\"\n", keychainService, keychainService, key))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("키체인에 저장하지 못했어요: %s", lastLine(string(out)))
		}
		if claudeKey(support) != key {
			return errors.New("키체인에 저장했지만 다시 읽지 못했어요")
		}
		return nil
	}
	return os.WriteFile(filepath.Join(support, "claude-key"), []byte(key), 0o600)
}

func (a *App) handleClaudeKey(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key string `json:"key"`
	}
	readJSON(r, &in)
	if err := setClaudeKey(a.support, strings.TrimSpace(in.Key)); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	writeJSON(w, a.claudeState())
}

func (a *App) handleClaudeTest(w http.ResponseWriter, r *http.Request) {
	key := claudeKey(a.support)
	if key == "" {
		jsonError(w, 400, "API 키가 아직 없어요")
		return
	}
	model := a.Settings().Model
	if model == "" {
		model = helperModels[0]["id"]
	}
	var resp struct {
		Usage struct {
			In  int `json:"input_tokens"`
			Out int `json:"output_tokens"`
		} `json:"usage"`
	}
	err := claudePost(key, map[string]any{"model": model, "max_tokens": 16,
		"messages": []map[string]any{{"role": "user", "content": "연결 시험이에요. '좋아요' 한 마디로 답해 주세요."}}}, 40*time.Second, &resp)
	if err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	addUsage(a.support, resp.Usage.In, resp.Usage.Out)
	writeJSON(w, map[string]any{"ok": true, "model": model})
}

func (a *App) claudeState() map[string]any {
	k := claudeKey(a.support)
	masked := ""
	if len(k) > 12 {
		masked = k[:7] + "…" + k[len(k)-4:]
	}
	return map[string]any{"hasKey": k != "", "masked": masked, "usage": monthUsage(a.support), "models": helperModels}
}

func (a *App) handleClaudeState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.claudeState())
}
