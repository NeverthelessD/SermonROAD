package main

// 설교 도우미가 부르는 AI — Claude(Anthropic) · ChatGPT(OpenAI) · 제미나이(Google)
//   · 셋 다 같은 요청문(helper.go 의 buildPrompt)과 같은 결과 모양(resultSchema)을 써요.
//       Claude  : Messages API + 꼭 써야 하는 도구(sermon_helper_result)
//       ChatGPT : Responses API + JSON 스키마(text.format)
//       제미나이 : generateContent + responseSchema (application/json)
//   · API 키는 AI 마다 따로, 맥 키체인 / Windows 자격 증명 관리자에만 저장해요.
//   · 모델 목록은 키가 있으면 그 AI 에서 직접 받아 와요 (새 모델이 나와도 앱을 고치지 않아도 되게). 못 받으면 아래 기본 목록.
//   · 시험용: SERMONROAD_CLAUDE_URL · SERMONROAD_OPENAI_URL · SERMONROAD_GEMINI_URL (주소 바꾸기), *_KEY (키 바꾸기)

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type aiProvider struct {
	ID         string // claude | chatgpt | gemini
	Name       string // 화면에 보이는 이름
	Company    string
	Service    string // 키체인 · 자격 증명 관리자 이름
	KeyFile    string // 리눅스 개발용
	EnvKey     string
	ChatURL    string // 구독 방식: 붙여 넣을 곳
	ConsoleURL string // 키 받는 곳
	Default    string
	Models     []map[string]string // 기본 목록 (추천 순)
}

var aiProviders = []*aiProvider{
	{ID: "claude", Name: "Claude", Company: "Anthropic", Service: "SermonROAD Claude API", KeyFile: "claude-key", EnvKey: "CLAUDE_KEY",
		ChatURL: "https://claude.ai/new", ConsoleURL: "https://console.anthropic.com/settings/keys", Default: "claude-sonnet-5-5",
		Models: []map[string]string{
			{"id": "claude-sonnet-5-5", "label": "Sonnet 5.5 — 추천 (빠르고 깊이 있음)"},
			{"id": "claude-opus-5-5", "label": "Opus 5.5 — 가장 깊이 (느리고 요금이 더 들어요)"},
			{"id": "claude-haiku-5-5", "label": "Haiku 5.5 — 가장 빠르고 저렴함"},
		}},
	{ID: "chatgpt", Name: "ChatGPT", Company: "OpenAI", Service: "SermonROAD OpenAI API", KeyFile: "openai-key", EnvKey: "OPENAI_KEY",
		ChatURL: "https://chatgpt.com/", ConsoleURL: "https://platform.openai.com/api-keys", Default: "gpt-6.1-sol",
		Models: []map[string]string{
			{"id": "gpt-6.1-sol", "label": "GPT-6.1 Sol — 추천 (깊이와 속도가 고름)"},
			{"id": "gpt-6-astra", "label": "GPT-6 Astra — 가장 깊이 (느리고 요금이 더 들어요)"},
			{"id": "gpt-6.1-terra", "label": "GPT-6.1 Terra — 중간"},
			{"id": "gpt-6-luna", "label": "GPT-6 Luna — 가장 빠르고 저렴함"},
		}},
	{ID: "gemini", Name: "제미나이", Company: "Google", Service: "SermonROAD Gemini API", KeyFile: "gemini-key", EnvKey: "GEMINI_KEY",
		ChatURL: "https://gemini.google.com/app", ConsoleURL: "https://aistudio.google.com/apikey", Default: "gemini-3.8-flash",
		Models: []map[string]string{
			{"id": "gemini-3.8-flash", "label": "Gemini 3.8 Flash — 추천 (빠르고 저렴함 · 무료 구간 있음)"},
			{"id": "gemini-3.1-pro-preview", "label": "Gemini 3.1 Pro — 가장 깊이 (미리보기 모델)"},
			{"id": "gemini-3.5-flash-lite", "label": "Gemini 3.5 Flash-Lite — 가장 가벼움"},
		}},
}

func providerByID(id string) *aiProvider {
	for _, p := range aiProviders {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func validAI(id string) string {
	if providerByID(id) != nil {
		return id
	}
	return "claude"
}

// 이번에 쓸 AI: 요청에 적힌 것 → 설정에서 고른 것
func (a *App) provider(id string) *aiProvider {
	if p := providerByID(id); p != nil {
		return p
	}
	return providerByID(validAI(a.Settings().AI))
}

// 이번에 쓸 모델: 설정 → (받아 온 목록에 없으면) 목록의 첫 추천
func (a *App) modelFor(id string) string {
	p := providerByID(validAI(id))
	m := a.Settings().Models[p.ID]
	if m == "" {
		m = p.Default
	}
	if live := cachedModels(p.ID, aiKey(a.support, p.ID)); len(live) > 0 && !hasModel(live, m) {
		return live[0]["id"]
	}
	return m
}

func hasModel(list []map[string]string, id string) bool {
	for _, x := range list {
		if x["id"] == id {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- 결과 모양 (세 AI 공통)

var resultSchema = map[string]any{
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
}

const resultName = "sermon_helper_result"

// 제미나이의 responseSchema 는 OpenAPI 모양이에요 (type 을 대문자로, description 은 그대로)
func geminiSchema(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			if k == "type" {
				out[k] = strings.ToUpper(fmt.Sprint(val))
			} else {
				out[k] = geminiSchema(val)
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = geminiSchema(x[i])
		}
		return out
	}
	return v
}

// ---------------------------------------------------------------- 부르기

func aiURL(id string) string {
	switch id {
	case "chatgpt":
		if u := env("OPENAI_URL"); u != "" {
			return strings.TrimRight(u, "/")
		}
		return "https://api.openai.com/v1"
	case "gemini":
		if u := env("GEMINI_URL"); u != "" {
			return strings.TrimRight(u, "/")
		}
		return "https://generativelanguage.googleapis.com/v1beta"
	}
	if u := env("CLAUDE_URL"); u != "" {
		return u // 예전처럼 messages 주소 그대로
	}
	return "https://api.anthropic.com/v1/messages"
}

func callAI(support string, p *aiProvider, key, model string, bp *builtPrompt) (map[string]any, error) {
	if model == "" {
		model = p.Default
	}
	var res map[string]any
	var in, out int
	var cut bool
	var err error
	switch p.ID {
	case "chatgpt":
		res, in, out, cut, err = callOpenAI(key, model, bp)
	case "gemini":
		res, in, out, cut, err = callGemini(key, model, bp)
	default:
		res, in, out, cut, err = callClaude(key, model, bp)
	}
	if in+out > 0 {
		addUsage(support, p.ID, in, out)
	}
	if err != nil {
		return nil, err
	}
	if cut {
		res["caution"] = strings.TrimSpace(strings.TrimPrefix(fmt.Sprint(res["caution"]), "<nil>") + " 답이 길어서 끝부분이 잘렸을 수 있어요.")
	}
	if _, ok := res["items"].([]any); !ok {
		return nil, fmt.Errorf("%s의 답을 읽지 못했어요. 다시 시도해 주세요", nameGap(p.Name))
	}
	return res, nil
}

func callClaude(key, model string, bp *builtPrompt) (map[string]any, int, int, bool, error) {
	body := map[string]any{
		"model":       model,
		"max_tokens":  8000,
		"system":      bp.System,
		"messages":    []map[string]any{{"role": "user", "content": bp.User}},
		"tools":       []any{map[string]any{"name": resultName, "description": "설교 도우미의 정리 결과를 돌려줍니다.", "input_schema": resultSchema}},
		"tool_choice": map[string]any{"type": "tool", "name": resultName},
	}
	var resp struct {
		Content []struct {
			Type  string         `json:"type"`
			Input map[string]any `json:"input"`
		} `json:"content"`
		Usage struct {
			In  int `json:"input_tokens"`
			Out int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}
	if err := aiPost(providerByID("claude"), aiURL("claude"), claudeHeaders(key), body, 6*time.Minute, &resp); err != nil {
		return nil, 0, 0, false, err
	}
	for _, c := range resp.Content {
		if c.Type == "tool_use" && c.Input != nil {
			return c.Input, resp.Usage.In, resp.Usage.Out, resp.StopReason == "max_tokens", nil
		}
	}
	return nil, resp.Usage.In, resp.Usage.Out, false, errors.New("Claude 의 답을 읽지 못했어요. 다시 시도해 주세요")
}

func claudeHeaders(key string) map[string]string {
	return map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"}
}

func callOpenAI(key, model string, bp *builtPrompt) (map[string]any, int, int, bool, error) {
	body := map[string]any{
		"model":             model,
		"instructions":      bp.System,
		"input":             bp.User,
		"max_output_tokens": 16000, // 생각하는 데 쓰는 토큰까지 들어가서 넉넉히
		"store":             false,
		"text": map[string]any{"format": map[string]any{
			"type": "json_schema", "name": resultName, "schema": resultSchema, "strict": false,
		}},
	}
	var resp struct {
		Status     string `json:"status"`
		Incomplete struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			In  int `json:"input_tokens"`
			Out int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := aiPost(providerByID("chatgpt"), aiURL("chatgpt")+"/responses", map[string]string{"authorization": "Bearer " + key}, body, 8*time.Minute, &resp); err != nil {
		return nil, 0, 0, false, err
	}
	var text strings.Builder
	for _, o := range resp.Output {
		if o.Type != "message" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "output_text" {
				text.WriteString(c.Text)
			} else if c.Refusal != "" {
				return nil, resp.Usage.In, resp.Usage.Out, false, errors.New("ChatGPT 가 이 요청에 답하지 않았어요: " + c.Refusal)
			}
		}
	}
	cut := resp.Status == "incomplete" && resp.Incomplete.Reason == "max_output_tokens"
	res, err := parseJSONResult(text.String(), "ChatGPT", cut)
	return res, resp.Usage.In, resp.Usage.Out, cut, err
}

func callGemini(key, model string, bp *builtPrompt) (map[string]any, int, int, bool, error) {
	body := map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": bp.System}}},
		"contents":          []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": bp.User}}}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseSchema":   geminiSchema(resultSchema),
			"maxOutputTokens":  16000,
		},
	}
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
		Usage struct {
			In      int `json:"promptTokenCount"`
			Out     int `json:"candidatesTokenCount"`
			Thought int `json:"thoughtsTokenCount"`
		} `json:"usageMetadata"`
	}
	u := aiURL("gemini") + "/models/" + strings.TrimPrefix(model, "models/") + ":generateContent"
	if err := aiPost(providerByID("gemini"), u, map[string]string{"x-goog-api-key": key}, body, 8*time.Minute, &resp); err != nil {
		return nil, 0, 0, false, err
	}
	in, out := resp.Usage.In, resp.Usage.Out+resp.Usage.Thought
	if resp.PromptFeedback.BlockReason != "" || len(resp.Candidates) == 0 {
		return nil, in, out, false, errors.New("제미나이가 이 요청에 답하지 않았어요. 자료를 조금 바꿔 다시 시도해 주세요")
	}
	c := resp.Candidates[0]
	var text strings.Builder
	for _, p := range c.Content.Parts {
		if !p.Thought {
			text.WriteString(p.Text)
		}
	}
	cut := c.FinishReason == "MAX_TOKENS"
	if text.Len() == 0 && c.FinishReason != "" && c.FinishReason != "STOP" && !cut {
		return nil, in, out, false, errors.New("제미나이가 답을 끝내지 못했어요 (" + c.FinishReason + "). 다시 시도해 주세요")
	}
	res, err := parseJSONResult(text.String(), "제미나이", cut)
	return res, in, out, cut, err
}

// JSON 으로 받은 답 읽기 (혹시 ```json … ``` 으로 감싸 와도 읽어요)
func parseJSONResult(t, name string, cut bool) (map[string]any, error) {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(t, "```json")
	t = strings.TrimPrefix(t, "```")
	t = strings.TrimSuffix(t, "```")
	t = strings.TrimSpace(t)
	if i := strings.Index(t, "{"); i > 0 {
		t = t[i:]
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(t), &res); err != nil {
		if cut {
			return nil, fmt.Errorf("%s의 답이 너무 길어서 끝까지 받지 못했어요. 자료를 조금 줄여 다시 시도해 주세요", nameGap(name))
		}
		return nil, fmt.Errorf("%s의 답을 읽지 못했어요. 다시 시도해 주세요", nameGap(name))
	}
	return res, nil
}

func aiPost(p *aiProvider, url string, headers map[string]string, body map[string]any, timeout time.Duration, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	return aiDo(p, req, headers, timeout, out)
}

func aiDo(p *aiProvider, req *http.Request, headers map[string]string, timeout time.Duration, out any) error {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "SermonROAD/"+build)
	res, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "Timeout") || strings.Contains(err.Error(), "deadline") {
			return fmt.Errorf("%s의 답이 너무 오래 걸려요. 자료를 조금 줄이거나 잠시 뒤 다시 시도해 주세요", nameGap(p.Name))
		}
		return fmt.Errorf("%s에 연결하지 못했어요. 인터넷 연결을 확인해 주세요", nameGap(p.Name))
	}
	defer res.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != 200 {
		return aiError(p, res.StatusCode, rb)
	}
	if err := json.Unmarshal(rb, out); err != nil {
		return fmt.Errorf("%s의 답을 읽지 못했어요", nameGap(p.Name))
	}
	return nil
}

// 세 AI 의 오류를 목사님이 알아듣기 쉬운 말로
func aiError(p *aiProvider, code int, rb []byte) error {
	var e struct {
		Error struct {
			Type    string `json:"type"`
			Code    any    `json:"code"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(rb, &e)
	msg := e.Error.Message
	lm := strings.ToLower(msg + " " + e.Error.Type + " " + fmt.Sprint(e.Error.Code) + " " + e.Error.Status)
	where := p.Name + " 연결"
	switch {
	case code == 401 || strings.Contains(lm, "api key not valid") || strings.Contains(lm, "invalid_api_key") || strings.Contains(lm, "api_key_invalid"):
		return fmt.Errorf("API 키가 맞지 않아요. 설정 › %s 에서 키를 다시 넣어 주세요", where)
	case strings.Contains(lm, "insufficient_quota") || strings.Contains(lm, "credit") || strings.Contains(lm, "billing"):
		return fmt.Errorf("API 사용 금액(크레딧)이 모자라요. %s 의 결제 화면에서 충전해 주세요", p.Company)
	case code == 404 || strings.Contains(lm, "model_not_found") || (strings.Contains(lm, "model") && (strings.Contains(lm, "not found") || strings.Contains(lm, "does not exist") || strings.Contains(lm, "not supported"))):
		return fmt.Errorf("고른 모델을 쓸 수 없어요. 설정 › %s 에서 다른 모델을 골라 주세요", where)
	case code == 403:
		return fmt.Errorf("이 API 키로는 쓸 수 없어요 (권한 없음). %s 에서 키를 확인해 주세요", p.Company)
	case code == 429 && p.ID == "gemini":
		return errors.New("제미나이 사용 한도를 넘었어요 (무료 구간은 1분 · 하루 한도가 있어요). 잠시 뒤 다시 시도하거나 다른 AI 로 받아 보세요")
	case code == 429:
		return errors.New("잠깐 사이에 요청이 많았어요. 1분쯤 뒤 다시 시도해 주세요")
	case code == 529 || code >= 500:
		return fmt.Errorf("%s가 지금 바빠요. 잠시 뒤 다시 시도해 주세요", nameGap(p.Name))
	case strings.Contains(lm, "too long") || strings.Contains(lm, "context length") || strings.Contains(lm, "context_length") || strings.Contains(lm, "token count") || strings.Contains(lm, "exceeds the maximum"):
		return errors.New("보낼 자료가 너무 많아요. 자료를 몇 개 빼고 다시 시도해 주세요")
	}
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", code)
	}
	return fmt.Errorf("%s가 요청을 받지 않았어요: %s", p.Name, msg)
}

// ---------------------------------------------------------------- 모델 목록 (키가 있으면 그 AI 에서 받아 와요)

var liveModels = struct {
	sync.Mutex
	m map[string]liveList
}{m: map[string]liveList{}}

type liveList struct {
	at   time.Time
	key  string
	list []map[string]string
}

func keyTag(key string) string {
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h[:6])
}

func cachedModels(id, key string) []map[string]string {
	if key == "" {
		return nil
	}
	liveModels.Lock()
	defer liveModels.Unlock()
	if l, ok := liveModels.m[id]; ok && l.key == keyTag(key) && time.Since(l.at) < 6*time.Hour {
		return l.list
	}
	return nil
}

var (
	reGPT       = regexp.MustCompile(`^gpt-\d`)
	reGPTSkip   = regexp.MustCompile(`audio|realtime|image|transcribe|tts|search|live|cyber|codex|embed|moder|instruct|translate|whisper|preview-\d|-\d{4}-\d{2}-\d{2}$`)
	reGemini    = regexp.MustCompile(`^gemini-\d`)
	reGemSkip   = regexp.MustCompile(`image|tts|embed|live|audio|robotics|computer|native|vision|aqa|-exp|learnlm|-\d{3}$`)
	reVersionNo = regexp.MustCompile(`\d+(\.\d+)?`)
)

// 그 AI 에서 쓸 수 있는 글 모델을 받아 와요. 추천(기본 목록)에 있는 것을 먼저, 나머지는 새 버전부터.
func fetchModels(p *aiProvider, key string) ([]map[string]string, error) {
	if l := cachedModels(p.ID, key); l != nil {
		return l, nil
	}
	var ids []string
	names := map[string]string{}
	switch p.ID {
	case "chatgpt":
		req, _ := http.NewRequest("GET", aiURL("chatgpt")+"/models", nil)
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := aiDo(p, req, map[string]string{"authorization": "Bearer " + key}, 30*time.Second, &resp); err != nil {
			return nil, err
		}
		for _, d := range resp.Data {
			if reGPT.MatchString(d.ID) && !reGPTSkip.MatchString(d.ID) {
				ids = append(ids, d.ID)
			}
		}
	case "gemini":
		req, _ := http.NewRequest("GET", aiURL("gemini")+"/models?pageSize=1000", nil)
		var resp struct {
			Models []struct {
				Name    string   `json:"name"`
				Display string   `json:"displayName"`
				Methods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
		}
		if err := aiDo(p, req, map[string]string{"x-goog-api-key": key}, 30*time.Second, &resp); err != nil {
			return nil, err
		}
		for _, m := range resp.Models {
			id := strings.TrimPrefix(m.Name, "models/")
			ok := false
			for _, x := range m.Methods {
				if x == "generateContent" {
					ok = true
				}
			}
			if ok && reGemini.MatchString(id) && !reGemSkip.MatchString(id) {
				ids = append(ids, id)
				names[id] = m.Display
			}
		}
	default:
		req, _ := http.NewRequest("GET", strings.TrimSuffix(aiURL("claude"), "/messages")+"/models?limit=100", nil)
		var resp struct {
			Data []struct {
				ID      string `json:"id"`
				Display string `json:"display_name"`
			} `json:"data"`
		}
		if err := aiDo(p, req, claudeHeaders(key), 30*time.Second, &resp); err != nil {
			return nil, err
		}
		for _, d := range resp.Data {
			if strings.HasPrefix(d.ID, "claude-") {
				ids = append(ids, d.ID)
				names[d.ID] = d.Display
			}
		}
	}
	if len(ids) == 0 {
		return nil, errors.New("쓸 수 있는 모델을 찾지 못했어요")
	}
	have := map[string]bool{}
	for _, id := range ids {
		have[id] = true
	}
	var list []map[string]string
	for _, m := range p.Models {
		if have[m["id"]] {
			list = append(list, m)
			delete(have, m["id"])
		}
	}
	var rest []string
	for id := range have {
		rest = append(rest, id)
	}
	sort.Slice(rest, func(i, j int) bool { // 버전 숫자가 큰 것부터
		vi, vj := verNum(rest[i]), verNum(rest[j])
		if vi != vj {
			return vi > vj
		}
		return rest[i] < rest[j]
	})
	if len(rest) > 14 {
		rest = rest[:14]
	}
	for _, id := range rest {
		label := id
		if n := names[id]; n != "" && n != id {
			label = n + " (" + id + ")"
		}
		list = append(list, map[string]string{"id": id, "label": label})
	}
	liveModels.Lock()
	liveModels.m[p.ID] = liveList{at: time.Now(), key: keyTag(key), list: list}
	liveModels.Unlock()
	return list, nil
}

func verNum(id string) float64 {
	var f float64
	fmt.Sscanf(reVersionNo.FindString(id), "%g", &f)
	return f
}

// ---------------------------------------------------------------- 사용량 (이번 달, AI 마다)

var usageMu sync.Mutex

func usageKey(id string) string {
	k := time.Now().Format("2006-01")
	if id != "" && id != "claude" { // Claude 는 예전 기록과 같은 자리
		k += "/" + id
	}
	return k
}

func addUsage(support, id string, in, out int) {
	usageMu.Lock()
	defer usageMu.Unlock()
	p := filepath.Join(support, "usage.json")
	m := map[string]map[string]int{}
	if b, err := os.ReadFile(p); err == nil {
		json.Unmarshal(b, &m)
	}
	k := usageKey(id)
	if m[k] == nil {
		m[k] = map[string]int{}
	}
	m[k]["calls"]++
	m[k]["in"] += in
	m[k]["out"] += out
	b, _ := json.Marshal(m)
	writeFileAtomic(p, b, 0o644)
}

func monthUsage(support, id string) map[string]int {
	usageMu.Lock()
	defer usageMu.Unlock()
	m := map[string]map[string]int{}
	if b, err := os.ReadFile(filepath.Join(support, "usage.json")); err == nil {
		json.Unmarshal(b, &m)
	}
	if u := m[usageKey(id)]; u != nil {
		return u
	}
	return map[string]int{"calls": 0, "in": 0, "out": 0}
}

// ---------------------------------------------------------------- API 키 (맥 키체인 · Windows 자격 증명 관리자, AI 마다 따로)

var reKey = regexp.MustCompile(`^[A-Za-z0-9_\-.]{20,400}$`)

func aiKey(support, id string) string {
	p := providerByID(validAI(id))
	if k := env(p.EnvKey); k != "" {
		return k
	}
	if isWindows {
		return winCredGet(p.Service)
	}
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("/usr/bin/security", "find-generic-password", "-a", "SermonROAD", "-s", p.Service, "-w").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	b, _ := os.ReadFile(filepath.Join(support, p.KeyFile))
	return strings.TrimSpace(string(b))
}

// 다른 AI 의 키를 잘못 넣었는지
func keyOwner(key string) string {
	switch {
	case strings.HasPrefix(key, "sk-ant-"):
		return "claude"
	case strings.HasPrefix(key, "AIza"):
		return "gemini"
	case strings.HasPrefix(key, "sk-"):
		return "chatgpt"
	}
	return ""
}

func setAIKey(support, id, key string) error {
	p := providerByID(validAI(id))
	if key == "" {
		if isWindows {
			winCredDelete(p.Service)
		} else if runtime.GOOS == "darwin" {
			exec.Command("/usr/bin/security", "delete-generic-password", "-a", "SermonROAD", "-s", p.Service).Run()
		} else {
			os.Remove(filepath.Join(support, p.KeyFile))
		}
		liveModels.Lock()
		delete(liveModels.m, p.ID)
		liveModels.Unlock()
		return nil
	}
	if o := keyOwner(key); o != "" && o != p.ID {
		return fmt.Errorf("이건 %s의 키예요. %s를 고른 뒤 넣어 주세요", nameGap(providerByID(o).Name), nameGap(providerByID(o).Name))
	}
	if !reKey.MatchString(key) {
		return errors.New("API 키 모양이 아니에요. " + keyHint(p.ID) + " 키를 그대로 붙여 넣어 주세요")
	}
	if isWindows {
		if err := winCredSet(p.Service, key); err != nil {
			return fmt.Errorf("Windows 자격 증명 관리자에 저장하지 못했어요: %v", err)
		}
		if aiKey(support, p.ID) != key {
			return errors.New("자격 증명 관리자에 저장했지만 다시 읽지 못했어요")
		}
		return nil
	}
	if runtime.GOOS == "darwin" {
		// 키를 명령줄에 드러내지 않도록 security 의 입력으로 넘겨요
		cmd := exec.Command("/usr/bin/security", "-i")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("add-generic-password -U -a SermonROAD -s \"%s\" -l \"%s\" -w \"%s\"\n", p.Service, p.Service, key))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("키체인에 저장하지 못했어요: %s", lastLine(string(out)))
		}
		if aiKey(support, p.ID) != key {
			return errors.New("키체인에 저장했지만 다시 읽지 못했어요")
		}
		return nil
	}
	return os.WriteFile(filepath.Join(support, p.KeyFile), []byte(key), 0o600)
}

func keyHint(id string) string {
	switch id {
	case "chatgpt":
		return "sk- 로 시작하는"
	case "gemini":
		return "AIza 로 시작하는"
	}
	return "sk-ant- 로 시작하는"
}

func maskKey(k string) string {
	if len(k) <= 12 {
		return ""
	}
	n := 7
	if strings.HasPrefix(k, "sk-proj-") {
		n = 8
	}
	return k[:n] + "…" + k[len(k)-4:]
}

// ---------------------------------------------------------------- 화면용 (/api/claude · /api/ai)

func (a *App) aiState() map[string]any {
	s := a.Settings()
	cur := validAI(s.AI)
	var list []map[string]any
	var top map[string]any
	for _, p := range aiProviders {
		k := aiKey(a.support, p.ID)
		models := p.Models
		live := false
		if l := cachedModels(p.ID, k); l != nil {
			models, live = l, true
		}
		x := map[string]any{"id": p.ID, "name": p.Name, "company": p.Company, "hasKey": k != "", "masked": maskKey(k),
			"usage": monthUsage(a.support, p.ID), "models": models, "liveModels": live, "model": a.modelFor(p.ID),
			"chatURL": p.ChatURL, "consoleURL": p.ConsoleURL, "keyHint": keyHint(p.ID)}
		list = append(list, x)
		if p.ID == cur {
			top = x
		}
	}
	// 예전 화면 · 시험과 맞도록 지금 고른 AI 의 값을 위에도 둬요
	return map[string]any{"ai": cur, "providers": list, "hasKey": top["hasKey"], "masked": top["masked"], "usage": top["usage"], "models": top["models"], "name": top["name"]}
}

func (a *App) handleAIState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.aiState())
}

// 키가 있는 AI 의 모델 목록을 새로 받아 와요 (설정 화면을 열 때)
func (a *App) handleAIModels(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AI string `json:"ai"`
	}
	readJSON(r, &in)
	p := a.provider(in.AI)
	if k := aiKey(a.support, p.ID); k != "" {
		if _, err := fetchModels(p, k); err != nil {
			jsonError(w, 400, err.Error())
			return
		}
	}
	writeJSON(w, a.aiState())
}

func (a *App) handleAIKey(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AI  string `json:"ai"`
		Key string `json:"key"`
	}
	readJSON(r, &in)
	if in.AI == "" {
		in.AI = "claude" // 예전 화면 · 시험은 Claude 키만 보냈어요
	}
	if err := setAIKey(a.support, in.AI, strings.TrimSpace(in.Key)); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	writeJSON(w, a.aiState())
}

// 연결 시험: 모델 목록을 받아 보고(키 확인 · 요금 없음), 고른 모델이 있는지 봐요
func (a *App) handleAITest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AI string `json:"ai"`
	}
	readJSON(r, &in)
	p := a.provider(in.AI)
	key := aiKey(a.support, p.ID)
	if key == "" {
		jsonError(w, 400, p.Name+" API 키가 아직 없어요")
		return
	}
	liveModels.Lock()
	delete(liveModels.m, p.ID)
	liveModels.Unlock()
	list, err := fetchModels(p, key)
	if err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	model := a.modelFor(p.ID)
	writeJSON(w, map[string]any{"ok": true, "ai": p.ID, "name": p.Name, "model": model, "models": len(list), "state": a.aiState()})
}

// ‘Claude 가’ · ‘제미나이가’ — 이름이 한글로 끝나면 조사를 붙여 써요
func nameGap(n string) string {
	if r := []rune(n); len(r) > 0 && r[len(r)-1] >= '가' && r[len(r)-1] <= '힣' {
		return n
	}
	return n + " "
}
