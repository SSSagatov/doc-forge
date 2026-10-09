package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalyze(t *testing.T) {
	valid := `{"summary":"Summary","requirements":[],"entities":["Company"],"risks":[]}`
	for _, tc := range []struct {
		name, content, finish string
		status                int
		wantError             bool
	}{
		{"success", valid, "STOP", 200, false},
		{"quota", "secret upstream error", "", 429, true},
		{"truncated", valid, "MAX_TOKENS", 200, true},
		{"invalid JSON", `{`, "STOP", 200, true},
		{"missing fields", `{"summary":"Only summary"}`, "STOP", 200, true},
		{"wrong type", `{"summary":"x","requirements":{},"entities":[],"risks":[]}`, "STOP", 200, true},
		{"extra JSON", valid + valid, "STOP", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-goog-api-key") != "test-key" || r.URL.RawQuery != "" || r.URL.Path != "/gemini-test:generateContent" {
					t.Error("invalid authentication or endpoint")
				}
				var payload struct {
					Contents []struct {
						Parts []struct {
							Inline struct {
								Data string `json:"data"`
							} `json:"inlineData"`
						} `json:"parts"`
					} `json:"contents"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if len(payload.Contents) != 1 || len(payload.Contents[0].Parts) != 2 {
					t.Error("missing PDF")
					return
				}
				pdf, err := base64.StdEncoding.DecodeString(payload.Contents[0].Parts[1].Inline.Data)
				if err != nil || string(pdf) != "%PDF-test" {
					t.Error("PDF encoding")
				}
				w.WriteHeader(tc.status)
				if tc.status != 200 {
					_, _ = w.Write([]byte(tc.content))
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"finishReason": tc.finish, "content": map[string]any{"parts": []any{map[string]any{"text": tc.content}}}}}, "usageMetadata": map[string]int{"promptTokenCount": 12, "candidatesTokenCount": 8}})
			}))
			defer server.Close()
			client := &Client{key: "test-key", model: "gemini-test", endpoint: server.URL + "/", http: server.Client()}
			result, err := client.Analyze(context.Background(), []byte("%PDF-test"))
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v", err)
			}
			if err != nil && (strings.Contains(err.Error(), "test-key") || strings.Contains(err.Error(), "secret upstream")) {
				t.Fatal("secret leaked")
			}
			if !tc.wantError && (result.Summary == nil || *result.PromptTokens != 12 || string(result.Requirements) != "[]") {
				t.Fatal("incorrect result")
			}
		})
	}
}

func TestConfig(t *testing.T) {
	t.Setenv("API_KEY", "")
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("missing key accepted")
	}
	t.Setenv("API_KEY", "test-key")
	t.Setenv("MODEL", "gemini-2.5-flash")
	t.Setenv("RESPONSE_FORMAT", "json_object")
	if _, err := NewFromEnv(); err != nil {
		t.Fatal(err)
	}
}
