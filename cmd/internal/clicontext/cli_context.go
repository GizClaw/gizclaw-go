package clicontext

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GizClaw/gizclaw-go/cmd/internal/identity"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/GizClaw/gizclaw-go/pkg/giznet/giznoise"
	"github.com/goccy/go-yaml"
)

// ServerConfig holds the connection info for a remote server.
type ServerConfig struct {
	Address    string              `yaml:"address"`
	PublicKey  giznet.PublicKey    `yaml:"-"`
	CipherMode giznoise.CipherMode `yaml:"cipher-mode,omitempty"`
}

// Config is the per-cli-context configuration stored in config.yaml.
type Config struct {
	Server ServerConfig `yaml:"server"`
}

// CLIContext represents a loaded CLI context directory.
type CLIContext struct {
	Name    string
	Dir     string
	Config  Config
	KeyPair *giznet.KeyPair
}

// Load reads a CLI context from its directory.
func Load(dir string) (*CLIContext, error) {
	name := filepath.Base(dir)

	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("clicontext: read config: %w", err)
	}
	var keyCheck struct {
		Server map[string]any `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &keyCheck); err != nil {
		return nil, fmt.Errorf("clicontext: parse config: %w", err)
	}
	if _, ok := keyCheck.Server["private-key"]; ok {
		return nil, fmt.Errorf("clicontext: server.private-key is not supported; use server.public-key")
	}
	if _, ok := keyCheck.Server["identity-key"]; ok {
		return nil, fmt.Errorf("clicontext: server.identity-key is not supported; use server.public-key")
	}
	var raw struct {
		Server struct {
			Address    string              `yaml:"address"`
			PublicKey  giznet.PublicKey    `yaml:"public-key"`
			CipherMode giznoise.CipherMode `yaml:"cipher-mode"`
		} `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("clicontext: parse config: %w", err)
	}
	if raw.Server.PublicKey.IsZero() {
		return nil, fmt.Errorf("clicontext: missing server.public-key")
	}
	if err := validateCipherMode(raw.Server.CipherMode); err != nil {
		return nil, err
	}
	cfg := Config{
		Server: ServerConfig{
			Address:    raw.Server.Address,
			PublicKey:  raw.Server.PublicKey,
			CipherMode: raw.Server.CipherMode,
		},
	}

	kp, err := identity.Load(filepath.Join(dir, "identity.key"))
	if err != nil {
		return nil, fmt.Errorf("clicontext: load identity: %w", err)
	}

	return &CLIContext{Name: name, Dir: dir, Config: cfg, KeyPair: kp}, nil
}

// ServerPublicKey parses and returns the server's public key.
func (c *CLIContext) ServerPublicKey() (giznet.PublicKey, error) {
	return c.Config.Server.PublicKey, nil
}
