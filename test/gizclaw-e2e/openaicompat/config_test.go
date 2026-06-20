package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigUsesFixedE2EDefaults(t *testing.T) {
	cfg, err := loadConfig(nil)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.APIKey != "test" || cfg.BaseURL != "http://127.0.0.1:8081/v1" {
		t.Fatalf("api config = %#v", cfg)
	}
	if cfg.ModelID != "e2e-doubao-2-lite-chat" || cfg.TTSModelID != "e2e-tts" || cfg.ASRModelID != "e2e-asr" || cfg.VoiceID != "" {
		t.Fatalf("model config = %#v", cfg)
	}
}

func TestLoadConfigFlagsOverrideDefaults(t *testing.T) {
	cfg, err := loadConfig([]string{
		"--model", "flag-chat",
		"--tts-model", "flag-tts",
		"--asr-model", "flag-asr",
		"--voice", "flag-voice",
	})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.ModelID != "flag-chat" || cfg.TTSModelID != "flag-tts" || cfg.ASRModelID != "flag-asr" || cfg.VoiceID != "flag-voice" {
		t.Fatalf("flag override config = %#v", cfg)
	}
}

func TestLoadConfigUsesFileConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "model": "setup-chat",
  "tts_model": "setup-tts",
  "asr_model": "setup-asr",
  "voice": "setup-voice",
  "thinking": {"enabled": false},
  "timeout": "3s"
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := loadConfig([]string{"--config", path, "--model", "flag-chat"})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.ModelID != "flag-chat" || cfg.TTSModelID != "setup-tts" || cfg.ASRModelID != "setup-asr" || cfg.VoiceID != "setup-voice" {
		t.Fatalf("file config = %#v", cfg)
	}
	if cfg.Thinking.Enabled == nil || *cfg.Thinking.Enabled {
		t.Fatalf("thinking config = %#v, want enabled=false", cfg.Thinking)
	}
	if cfg.Timeout != 3*time.Second {
		t.Fatalf("timeout = %s", cfg.Timeout)
	}
}

func TestRunAgainstFakeOpenAICompatServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			var body struct {
				Stream   bool `json:"stream"`
				Thinking *struct {
					Enabled *bool `json:"enabled"`
				} `json:"thinking"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode chat request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.Thinking == nil || body.Thinking.Enabled == nil || *body.Thinking.Enabled {
				t.Errorf("thinking = %#v, want enabled=false", body.Thinking)
				http.Error(w, "missing thinking.enabled=false", http.StatusBadRequest)
				return
			}
			if body.Stream {
				w.Header().Set("Content-Type", "text/event-stream")
				writeSSE(w, map[string]any{
					"id":      "chatcmpl-stream",
					"object":  "chat.completion.chunk",
					"created": 0,
					"model":   "chat",
					"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": "小猫今天"}, "finish_reason": nil}},
				})
				writeSSE(w, map[string]any{
					"id":      "chatcmpl-stream",
					"object":  "chat.completion.chunk",
					"created": 0,
					"model":   "chat",
					"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": "开心跑步"}, "finish_reason": "stop"}},
				})
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
				return
			}
			writeJSON(w, map[string]any{
				"id":      "chatcmpl",
				"object":  "chat.completion",
				"created": 0,
				"model":   "chat",
				"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": "小猫今天开心跑步"}, "finish_reason": "stop"}},
			})
		case "/v1/audio/speech":
			var body struct {
				StreamFormat string `json:"stream_format"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode speech request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.StreamFormat == "sse" {
				w.Header().Set("Content-Type", "text/event-stream")
				writeSSE(w, map[string]any{"type": "speech.audio.delta", "audio": base64.StdEncoding.EncodeToString([]byte{1, 2, 3})})
				writeSSE(w, map[string]any{"type": "speech.audio.done", "done": true})
				return
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte{4, 5, 6})
		case "/v1/audio/transcriptions":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("parse transcription request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if strings.EqualFold(r.FormValue("stream"), "true") {
				w.Header().Set("Content-Type", "text/event-stream")
				writeSSE(w, map[string]any{"type": "transcript.text.delta", "delta": "小猫今天"})
				writeSSE(w, map[string]any{"type": "transcript.text.done", "text": "小猫今天开心跑步"})
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
				return
			}
			writeJSON(w, map[string]any{"text": "小猫今天开心跑步"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"thinking":{"enabled":false}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := run([]string{
		"--base-url", server.URL + "/v1",
		"--model", "chat",
		"--tts-model", "tts",
		"--asr-model", "asr",
		"--voice", "voice",
		"--config", configPath,
		"--output-dir", t.TempDir(),
	}); err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

func TestFirstVoiceID(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		want     string
		wantErr  string
		wantAuth string
	}{
		{
			name:     "success",
			status:   http.StatusOK,
			body:     `{"data":[{"id":"ignored","provider":{"kind":"openai"}},{"id":"voice-a","provider":{"kind":"volc-tenant"}}]}`,
			want:     "voice-a",
			wantAuth: "Bearer key",
		},
		{
			name:    "non-200",
			status:  http.StatusForbidden,
			body:    `{"error":"denied"}`,
			wantErr: "list voices status = 403",
		},
		{
			name:    "bad-json",
			status:  http.StatusOK,
			body:    `{`,
			wantErr: "decode voices response",
		},
		{
			name:    "missing-volc",
			status:  http.StatusOK,
			body:    `{"data":[{"id":"voice-a","provider":{"kind":"openai"}}]}`,
			wantErr: "no volc voice returned",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/voices" {
					t.Errorf("path = %s, want /v1/voices", r.URL.Path)
				}
				if got := r.URL.Query().Get("providerKind"); got != "volc-tenant" {
					t.Errorf("providerKind = %q", got)
				}
				if got := r.URL.Query().Get("limit"); got != "20" {
					t.Errorf("limit = %q", got)
				}
				if tc.wantAuth != "" && r.Header.Get("Authorization") != tc.wantAuth {
					t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			got, err := firstVoiceID(context.Background(), server.Client(), config{
				APIKey:  "key",
				BaseURL: server.URL + "/v1",
			})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("firstVoiceID() error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("firstVoiceID() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("firstVoiceID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTranscriptionAssertions(t *testing.T) {
	if err := assertTranscriptionMatches("exact", "小猫，今天开心跑步!", "小猫今天开心跑步"); err != nil {
		t.Fatalf("assertTranscriptionMatches() error = %v", err)
	}
	if err := assertTranscriptionMatches("empty", "", "text"); err == nil {
		t.Fatal("assertTranscriptionMatches(empty expected) error = nil")
	}
	if err := assertTranscriptionSimilar("similar", "小猫今天开心跑步", "小猫今天开心散步", 0.75); err != nil {
		t.Fatalf("assertTranscriptionSimilar() error = %v", err)
	}
	if err := assertTranscriptionSimilar("different", "abcdef", "uvwxyz", 0.5); err == nil {
		t.Fatal("assertTranscriptionSimilar(different) error = nil")
	}
}

func TestAudioContentTypeAndFilename(t *testing.T) {
	contentTypes := map[string]string{
		"speech.ogg":  "audio/ogg",
		"speech.OPUS": "audio/ogg",
		"speech.mp3":  "audio/mpeg",
		"speech.wav":  "audio/wav",
		"speech.flac": "audio/flac",
		"speech.m4a":  "audio/mp4",
		"speech.bin":  "application/octet-stream",
	}
	for path, want := range contentTypes {
		if got := audioContentType(path); got != want {
			t.Fatalf("audioContentType(%q) = %q, want %q", path, got, want)
		}
	}

	filenames := []struct {
		name        string
		filename    string
		contentType string
		data        []byte
		want        string
	}{
		{name: "content type with params", filename: "speech.tmp", contentType: "audio/mpeg; charset=binary", want: "speech.mp3"},
		{name: "ogg magic", filename: "speech.tmp", data: []byte("OggS\x00"), want: "speech.ogg"},
		{name: "id3 magic", filename: "speech.tmp", data: []byte("ID3\x04"), want: "speech.mp3"},
		{name: "empty base", filename: ".tmp", contentType: "audio/wav", want: "audio.wav"},
		{name: "unknown", filename: "speech.tmp", contentType: "application/octet-stream", data: []byte{1, 2, 3}, want: "speech.tmp"},
	}
	for _, tt := range filenames {
		t.Run(tt.name, func(t *testing.T) {
			if got := audioFilename(tt.filename, tt.contentType, tt.data); got != tt.want {
				t.Fatalf("audioFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAudioFilenameUsesActualFormat(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		data        []byte
		want        string
	}{
		{
			name:        "content type ogg overrides mp3 name",
			filename:    "speech.mp3",
			contentType: "audio/ogg; codecs=opus",
			want:        "speech.ogg",
		},
		{
			name:     "sniffs ogg",
			filename: "speech-stream.mp3",
			data:     []byte("OggS\x00\x02"),
			want:     "speech-stream.ogg",
		},
		{
			name:        "keeps mp3",
			filename:    "speech.mp3",
			contentType: "audio/mpeg",
			want:        "speech.mp3",
		},
		{
			name:     "unknown keeps original",
			filename: "speech.mp3",
			data:     []byte{1, 2, 3},
			want:     "speech.mp3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := audioFilename(tc.filename, tc.contentType, tc.data); got != tc.want {
				t.Fatalf("audioFilename() = %q, want %q", got, tc.want)
			}
		})
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeSSE(w http.ResponseWriter, value any) {
	data, _ := json.Marshal(value)
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n\n"))
}
