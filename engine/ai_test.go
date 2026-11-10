package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const fakeResult = `{"thesis":"하나님은 모든 것을 합력하여 선을 이루십니다","items":[{"head":"문맥","body":"8장의 흐름","sources":[{"id":"S1","page":313}],"outside":false},{"body":"덧붙임","outside":true}],"caution":""}`

// 세 AI 의 모양을 흉내 내는 모의 서버
func fakeAI(t *testing.T, seen map[string]map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body map[string]any
		json.Unmarshal(b, &body)
		switch {
		case r.URL.Path == "/claude/messages":
			seen["claude"] = body
			if r.Header.Get("x-api-key") != "sk-ant-good" {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
				return
			}
			var in map[string]any
			json.Unmarshal([]byte(fakeResult), &in)
			json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]any{"type": "tool_use", "name": resultName, "input": in}},
				"usage": map[string]any{"input_tokens": 100, "output_tokens": 50}, "stop_reason": "tool_use"})
		case r.URL.Path == "/openai/responses":
			seen["chatgpt"] = body
			if r.Header.Get("authorization") != "Bearer sk-proj-good" {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","code":"invalid_api_key"}}`))
				return
			}
			if body["model"] == "gpt-nope" {
				w.WriteHeader(404)
				w.Write([]byte(`{"error":{"message":"The model gpt-nope does not exist","code":"model_not_found"}}`))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"status": "completed", "output": []any{
				map[string]any{"type": "reasoning", "summary": []any{}},
				map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": fakeResult}}}},
				"usage": map[string]any{"input_tokens": 200, "output_tokens": 80}})
		case r.URL.Path == "/openai/models":
			json.NewEncoder(w).Encode(map[string]any{"data": []any{
				map[string]any{"id": "gpt-6-astra"}, map[string]any{"id": "gpt-6.1-sol"}, map[string]any{"id": "gpt-realtime-2"},
				map[string]any{"id": "gpt-7-nova"}, map[string]any{"id": "text-embedding-3-large"}, map[string]any{"id": "gpt-6-sol-2026-09-22"}}})
		case strings.HasPrefix(r.URL.Path, "/gemini/models/") && strings.HasSuffix(r.URL.Path, ":generateContent"):
			seen["gemini"] = body
			if r.Header.Get("x-goog-api-key") != "AIzaGood0000000000000000000000000000000" {
				w.WriteHeader(400)
				w.Write([]byte(`{"error":{"code":400,"message":"API key not valid. Please pass a valid API key.","status":"INVALID_ARGUMENT"}}`))
				return
			}
			if strings.Contains(r.URL.Path, "busy") {
				w.WriteHeader(429)
				w.Write([]byte(`{"error":{"code":429,"message":"quota","status":"RESOURCE_EXHAUSTED"}}`))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{
				map[string]any{"text": "생각 중…", "thought": true}, map[string]any{"text": "```json\n" + fakeResult + "\n```"}}}, "finishReason": "STOP"}},
				"usageMetadata": map[string]any{"promptTokenCount": 300, "candidatesTokenCount": 90, "thoughtsTokenCount": 10}})
		case r.URL.Path == "/gemini/models":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{
				map[string]any{"name": "models/gemini-3.8-flash", "displayName": "Gemini 3.8 Flash", "supportedGenerationMethods": []string{"generateContent"}},
				map[string]any{"name": "models/gemini-3.8-flash-image", "supportedGenerationMethods": []string{"generateContent"}},
				map[string]any{"name": "models/gemini-embedding-2", "supportedGenerationMethods": []string{"embedContent"}},
				map[string]any{"name": "models/gemini-4-pro", "displayName": "Gemini 4 Pro", "supportedGenerationMethods": []string{"generateContent"}}}})
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestCallAllAI(t *testing.T) {
	seen := map[string]map[string]any{}
	srv := fakeAI(t, seen)
	defer srv.Close()
	t.Setenv("SERMONROAD_CLAUDE_URL", srv.URL+"/claude/messages")
	t.Setenv("SERMONROAD_OPENAI_URL", srv.URL+"/openai")
	t.Setenv("SERMONROAD_GEMINI_URL", srv.URL+"/gemini")
	dir := t.TempDir()
	bp := &builtPrompt{System: "시스템", User: "본문: 롬 8:28"}
	keys := map[string]string{"claude": "sk-ant-good", "chatgpt": "sk-proj-good", "gemini": "AIzaGood0000000000000000000000000000000"}
	for _, p := range aiProviders {
		res, err := callAI(dir, p, keys[p.ID], p.Default, bp)
		if err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
		items, _ := res["items"].([]any)
		if len(items) != 2 || res["thesis"] == "" {
			t.Fatalf("%s: 결과 모양이 달라요 %v", p.ID, res)
		}
		if u := monthUsage(dir, p.ID); u["calls"] != 1 || u["in"] == 0 {
			t.Fatalf("%s: 사용량 %v", p.ID, u)
		}
		// 틀린 키
		if _, err := callAI(dir, p, "wrong-key-0000000000000000", p.Default, bp); err == nil || !strings.Contains(err.Error(), "API 키가 맞지 않아요") {
			t.Fatalf("%s: 틀린 키 오류가 달라요: %v", p.ID, err)
		}
	}
	// 요청 모양
	if seen["chatgpt"]["instructions"] != "시스템" || seen["chatgpt"]["text"].(map[string]any)["format"].(map[string]any)["type"] != "json_schema" {
		t.Fatalf("ChatGPT 요청 모양: %v", seen["chatgpt"])
	}
	gc := seen["gemini"]["generationConfig"].(map[string]any)
	if gc["responseMimeType"] != "application/json" || gc["responseSchema"].(map[string]any)["type"] != "OBJECT" {
		t.Fatalf("제미나이 요청 모양: %v", gc)
	}
	// 모델 없음 · 한도
	if _, err := callAI(dir, providerByID("chatgpt"), keys["chatgpt"], "gpt-nope", bp); err == nil || !strings.Contains(err.Error(), "다른 모델") {
		t.Fatalf("모델 없음: %v", err)
	}
	if _, err := callAI(dir, providerByID("gemini"), keys["gemini"], "gemini-busy", bp); err == nil || !strings.Contains(err.Error(), "한도") {
		t.Fatalf("제미나이 한도: %v", err)
	}
	// 모델 목록
	l, err := fetchModels(providerByID("chatgpt"), keys["chatgpt"])
	if err != nil || len(l) != 3 || l[0]["id"] != "gpt-6.1-sol" || l[2]["id"] != "gpt-7-nova" && l[1]["id"] != "gpt-6-astra" {
		t.Fatalf("ChatGPT 모델 목록: %v %v", l, err)
	}
	g, err := fetchModels(providerByID("gemini"), keys["gemini"])
	if err != nil || len(g) != 2 || g[0]["id"] != "gemini-3.8-flash" || g[1]["id"] != "gemini-4-pro" {
		t.Fatalf("제미나이 모델 목록: %v %v", g, err)
	}
}

func TestKeyChecks(t *testing.T) {
	dir := t.TempDir()
	if err := setAIKey(dir, "chatgpt", "sk-ant-api03-abcdefghijklmnopqrstuvwxyz"); err == nil || !strings.Contains(err.Error(), "Claude 의 키") {
		t.Fatalf("Claude 키를 ChatGPT 에: %v", err)
	}
	if err := setAIKey(dir, "claude", "AIzaSyAbcdefghijklmnopqrstuvwxyz0123456"); err == nil || !strings.Contains(err.Error(), "제미나이") {
		t.Fatalf("제미나이 키를 Claude 에: %v", err)
	}
	if err := setAIKey(dir, "gemini", "AIzaSyAbcdefghijklmnopqrstuvwxyz0123456"); err != nil {
		t.Fatal(err)
	}
	if aiKey(dir, "gemini") == "" || aiKey(dir, "claude") != "" {
		t.Fatal("키가 AI 마다 따로 저장되지 않았어요")
	}
	if m := maskKey("sk-proj-abcdefghijklmnop1234"); m != "sk-proj-…1234" {
		t.Fatal(m)
	}
}
