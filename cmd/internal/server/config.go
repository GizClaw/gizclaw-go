package server

import (
	"fmt"
	"os"

	"github.com/GizClaw/gizclaw-go/cmd/internal/storage"
	"github.com/GizClaw/gizclaw-go/cmd/internal/stores"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/goccy/go-yaml"
)

type Config struct {
	KeyPair        *giznet.KeyPair
	ListenAddr     string
	AdminPublicKey giznet.PublicKey
	Storage        map[string]storage.Config
	Stores         map[string]stores.Config
	Peers          PeersConfig
	Credentials    CredentialsConfig
	Firmwares      FirmwaresConfig
	MiniMax        MiniMaxConfig
	Workspaces     WorkspacesConfig
	Workflows      WorkflowsConfig
	ACL            ACLConfig
}

type PeersConfig struct {
	Store string `yaml:"store"`
}

type CredentialsConfig struct {
	Store string `yaml:"store"`
}

type FirmwaresConfig struct {
	Store string `yaml:"store"`
}

type MiniMaxConfig struct {
	TenantsStore     string `yaml:"tenants-store"`
	VoicesStore      string `yaml:"voices-store"`
	CredentialsStore string `yaml:"credentials-store"`
}

type WorkspacesConfig struct {
	Store string `yaml:"store"`
}

type WorkflowsConfig struct {
	Store string `yaml:"store"`
}

type ACLConfig struct {
	Store string `yaml:"store"`
}

type ConfigFile struct {
	ListenAddr     string                    `yaml:"listen"`
	AdminPublicKey giznet.PublicKey          `yaml:"admin-public-key"`
	Storage        map[string]storage.Config `yaml:"storage"`
	Stores         map[string]stores.Config  `yaml:"stores"`
	Peers          PeersConfig               `yaml:"peers"`
	Credentials    CredentialsConfig         `yaml:"credentials"`
	Firmwares      FirmwaresConfig           `yaml:"firmwares"`
	MiniMax        MiniMaxConfig             `yaml:"minimax"`
	Workspaces     WorkspacesConfig          `yaml:"workspaces"`
	Workflows      WorkflowsConfig           `yaml:"workflows"`
	ACL            ACLConfig                 `yaml:"acl"`
}

func LoadConfig(path string) (ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ConfigFile{}, err
	}
	var raw struct {
		ListenAddr     string                    `yaml:"listen"`
		AdminPublicKey *giznet.PublicKey         `yaml:"admin-public-key"`
		Storage        map[string]storage.Config `yaml:"storage"`
		Stores         map[string]stores.Config  `yaml:"stores"`
		Peers          PeersConfig               `yaml:"peers"`
		Credentials    CredentialsConfig         `yaml:"credentials"`
		Firmwares      FirmwaresConfig           `yaml:"firmwares"`
		MiniMax        MiniMaxConfig             `yaml:"minimax"`
		Workspaces     WorkspacesConfig          `yaml:"workspaces"`
		Workflows      WorkflowsConfig           `yaml:"workflows"`
		ACL            ACLConfig                 `yaml:"acl"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ConfigFile{}, err
	}
	var adminPublicKey giznet.PublicKey
	if raw.AdminPublicKey != nil {
		if raw.AdminPublicKey.IsZero() {
			return ConfigFile{}, fmt.Errorf("server: invalid admin-public-key: zero key")
		}
		adminPublicKey = *raw.AdminPublicKey
	}
	cfg := ConfigFile{
		ListenAddr:     raw.ListenAddr,
		AdminPublicKey: adminPublicKey,
		Storage:        raw.Storage,
		Stores:         raw.Stores,
		Peers:          raw.Peers,
		Credentials:    raw.Credentials,
		Firmwares:      raw.Firmwares,
		MiniMax:        raw.MiniMax,
		Workspaces:     raw.Workspaces,
		Workflows:      raw.Workflows,
		ACL:            raw.ACL,
	}
	return cfg, nil
}

func DefaultConfig() Config {
	return Config{
		ListenAddr: ":9820",
	}
}

func mergeFileConfig(cfg Config, fileCfg ConfigFile) (Config, error) {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = fileCfg.ListenAddr
	}
	if cfg.AdminPublicKey.IsZero() {
		cfg.AdminPublicKey = fileCfg.AdminPublicKey
	}
	if len(cfg.Stores) == 0 {
		cfg.Stores = fileCfg.Stores
	}
	if len(cfg.Storage) == 0 {
		cfg.Storage = fileCfg.Storage
	}
	cfg.Peers = mergePeersConfig(cfg.Peers, fileCfg.Peers)
	cfg.Credentials = mergeCredentialsConfig(cfg.Credentials, fileCfg.Credentials)
	cfg.Firmwares = mergeFirmwaresConfig(cfg.Firmwares, fileCfg.Firmwares)
	cfg.MiniMax = mergeMiniMaxConfig(cfg.MiniMax, fileCfg.MiniMax)
	cfg.Workspaces = mergeWorkspacesConfig(cfg.Workspaces, fileCfg.Workspaces)
	cfg.Workflows = mergeWorkflowsConfig(cfg.Workflows, fileCfg.Workflows)
	cfg.ACL = mergeACLConfig(cfg.ACL, fileCfg.ACL)
	return cfg, nil
}

func mergePeersConfig(runtime PeersConfig, file PeersConfig) PeersConfig {
	if runtime.Store == "" {
		runtime.Store = file.Store
	}
	return runtime
}

func mergeCredentialsConfig(runtime CredentialsConfig, file CredentialsConfig) CredentialsConfig {
	if runtime.Store == "" {
		runtime.Store = file.Store
	}
	return runtime
}

func mergeFirmwaresConfig(runtime FirmwaresConfig, file FirmwaresConfig) FirmwaresConfig {
	if runtime.Store == "" {
		runtime.Store = file.Store
	}
	return runtime
}

func mergeMiniMaxConfig(runtime MiniMaxConfig, file MiniMaxConfig) MiniMaxConfig {
	if runtime.TenantsStore == "" {
		runtime.TenantsStore = file.TenantsStore
	}
	if runtime.VoicesStore == "" {
		runtime.VoicesStore = file.VoicesStore
	}
	if runtime.CredentialsStore == "" {
		runtime.CredentialsStore = file.CredentialsStore
	}
	return runtime
}

func mergeWorkspacesConfig(runtime WorkspacesConfig, file WorkspacesConfig) WorkspacesConfig {
	if runtime.Store == "" {
		runtime.Store = file.Store
	}
	return runtime
}

func mergeWorkflowsConfig(runtime WorkflowsConfig, file WorkflowsConfig) WorkflowsConfig {
	if runtime.Store == "" {
		runtime.Store = file.Store
	}
	return runtime
}

func mergeACLConfig(runtime ACLConfig, file ACLConfig) ACLConfig {
	if runtime.Store == "" {
		runtime.Store = file.Store
	}
	return runtime
}

func prepareConfig(cfg Config) (Config, error) {
	defaults := DefaultConfig()
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaults.ListenAddr
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	if cfg.KeyPair == nil {
		keyPair, err := giznet.GenerateKeyPair()
		if err != nil {
			return Config{}, fmt.Errorf("server: generate key pair: %w", err)
		}
		cfg.KeyPair = keyPair
	}
	return cfg, nil
}

func (cfg Config) validate() error {
	if cfg.Peers.Store == "" {
		return fmt.Errorf("server: peers.store is required")
	}
	if len(cfg.Storage) == 0 {
		return nil
	}
	if cfg.Credentials.Store == "" {
		return fmt.Errorf("server: credentials.store is required")
	}
	if cfg.Firmwares.Store == "" {
		return fmt.Errorf("server: firmwares.store is required")
	}
	if cfg.MiniMax.TenantsStore == "" {
		return fmt.Errorf("server: minimax.tenants-store is required")
	}
	if cfg.MiniMax.VoicesStore == "" {
		return fmt.Errorf("server: minimax.voices-store is required")
	}
	if cfg.MiniMax.CredentialsStore == "" {
		return fmt.Errorf("server: minimax.credentials-store is required")
	}
	if cfg.Workspaces.Store == "" {
		return fmt.Errorf("server: workspaces.store is required")
	}
	if cfg.Workflows.Store == "" {
		return fmt.Errorf("server: workflows.store is required")
	}
	if cfg.ACL.Store == "" {
		return fmt.Errorf("server: acl.store is required")
	}
	return nil
}
