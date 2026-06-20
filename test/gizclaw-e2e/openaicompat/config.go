package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type config struct {
	APIKey     string
	BaseURL    string
	ConfigPath string
	ModelID    string
	TTSModelID string
	ASRModelID string
	VoiceID    string
	Thinking   thinkingConfig
	OutputDir  string
	Timeout    time.Duration
}

type fileConfig struct {
	APIKey    string          `json:"api_key"`
	BaseURL   string          `json:"base_url"`
	Model     string          `json:"model"`
	TTSModel  string          `json:"tts_model"`
	ASRModel  string          `json:"asr_model"`
	Voice     string          `json:"voice"`
	Thinking  *thinkingConfig `json:"thinking"`
	OutputDir string          `json:"output_dir"`
	Timeout   string          `json:"timeout"`
}

type thinkingConfig struct {
	Enabled *bool  `json:"enabled"`
	Level   string `json:"level"`
}

func loadConfig(args []string) (config, error) {
	cfg := config{
		APIKey:     "test",
		BaseURL:    "http://127.0.0.1:8081/v1",
		ModelID:    "e2e-doubao-2-lite-chat",
		TTSModelID: "e2e-tts",
		ASRModelID: "e2e-asr",
		Timeout:    90 * time.Second,
	}
	if configPath := findStringFlag(args, "config"); configPath != "" {
		cfg.ConfigPath = configPath
		if err := applyFileConfig(&cfg, configPath); err != nil {
			return config{}, err
		}
	}
	flags := flag.NewFlagSet("openai-compat", flag.ContinueOnError)
	flags.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "OpenAI-compatible e2e config JSON")
	flags.StringVar(&cfg.APIKey, "api-key", cfg.APIKey, "OpenAI-compatible API key")
	flags.StringVar(&cfg.BaseURL, "base-url", cfg.BaseURL, "OpenAI-compatible API base URL")
	flags.StringVar(&cfg.ModelID, "model", cfg.ModelID, "chat model id")
	flags.StringVar(&cfg.TTSModelID, "tts-model", cfg.TTSModelID, "speech model id")
	flags.StringVar(&cfg.ASRModelID, "asr-model", cfg.ASRModelID, "ASR model id")
	flags.StringVar(&cfg.VoiceID, "voice", cfg.VoiceID, "voice id; defaults to a synced Volc voice from /voices")
	flags.StringVar(&cfg.OutputDir, "output-dir", cfg.OutputDir, "directory for generated audio files")
	flags.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "request timeout")
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if cfg.Timeout <= 0 {
		return config{}, fmt.Errorf("timeout must be positive")
	}
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.ModelID = strings.TrimSpace(cfg.ModelID)
	cfg.TTSModelID = strings.TrimSpace(cfg.TTSModelID)
	cfg.ASRModelID = strings.TrimSpace(cfg.ASRModelID)
	cfg.VoiceID = strings.TrimSpace(cfg.VoiceID)
	cfg.OutputDir = strings.TrimSpace(cfg.OutputDir)
	if cfg.APIKey == "" {
		return config{}, fmt.Errorf("api key must not be empty")
	}
	if cfg.BaseURL == "" {
		return config{}, fmt.Errorf("base url must not be empty")
	}
	if cfg.ModelID == "" {
		return config{}, fmt.Errorf("chat model must be set with --model")
	}
	if cfg.TTSModelID == "" {
		return config{}, fmt.Errorf("speech model must be set with --tts-model")
	}
	if cfg.ASRModelID == "" {
		return config{}, fmt.Errorf("ASR model must be set with --asr-model")
	}
	if cfg.OutputDir == "" {
		var err error
		cfg.OutputDir, err = os.MkdirTemp("", "gizclaw-openai-compat-*")
		if err != nil {
			return config{}, fmt.Errorf("create output dir: %w", err)
		}
	}
	return cfg, nil
}

func findStringFlag(args []string, name string) string {
	long := "--" + name
	short := "-" + name
	var value string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if (arg == long || arg == short) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			value = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, long+"=") {
			value = strings.TrimPrefix(arg, long+"=")
			continue
		}
		if strings.HasPrefix(arg, short+"=") {
			value = strings.TrimPrefix(arg, short+"=")
		}
	}
	return strings.TrimSpace(value)
}

func applyFileConfig(cfg *config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	var fileCfg fileConfig
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("decode config %s: %w", path, err)
	}
	if fileCfg.APIKey != "" {
		cfg.APIKey = fileCfg.APIKey
	}
	if fileCfg.BaseURL != "" {
		cfg.BaseURL = fileCfg.BaseURL
	}
	if fileCfg.Model != "" {
		cfg.ModelID = fileCfg.Model
	}
	if fileCfg.TTSModel != "" {
		cfg.TTSModelID = fileCfg.TTSModel
	}
	if fileCfg.ASRModel != "" {
		cfg.ASRModelID = fileCfg.ASRModel
	}
	if fileCfg.Voice != "" {
		cfg.VoiceID = fileCfg.Voice
	}
	if fileCfg.Thinking != nil {
		cfg.Thinking = *fileCfg.Thinking
	}
	if fileCfg.OutputDir != "" {
		cfg.OutputDir = fileCfg.OutputDir
	}
	if fileCfg.Timeout != "" {
		timeout, err := time.ParseDuration(fileCfg.Timeout)
		if err != nil {
			return fmt.Errorf("config timeout: %w", err)
		}
		cfg.Timeout = timeout
	}
	return nil
}
