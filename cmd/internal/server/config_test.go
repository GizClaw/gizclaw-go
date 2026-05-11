package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/cmd/internal/storage"
	"github.com/GizClaw/gizclaw-go/cmd/internal/stores"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ListenAddr != ":9820" {
		t.Fatalf("ListenAddr = %q", cfg.ListenAddr)
	}
}

func TestNewWithLayeredStorageConfig(t *testing.T) {
	dir := t.TempDir()
	srv, err := New(validLayeredConfig(dir))
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	if srv.PeerStore == nil || srv.CredentialStore == nil || srv.MiniMaxTenantStore == nil || srv.VoiceStore == nil || srv.WorkspaceStore == nil || srv.WorkflowStore == nil || srv.DepotMetadataStore == nil {
		t.Fatalf("module stores not wired: %+v", srv)
	}
	if srv.DepotStore == nil {
		t.Fatal("DepotStore is nil")
	}
}

func TestNewWithLayeredStorageReportsStoreErrors(t *testing.T) {
	dir := t.TempDir()

	storageErrCfg := validLayeredConfig(dir)
	storageErrCfg.Storage["memory"] = storage.Config{Kind: storage.KindKeyValue, Backend: "redis"}
	if _, err := New(storageErrCfg); err == nil || !strings.Contains(err.Error(), "server: stores:") {
		t.Fatalf("New(storage error) = %v", err)
	}

	logicalErrCfg := validLayeredConfig(dir)
	logicalErrCfg.Stores["credentials"] = stores.Config{Kind: stores.KindKeyValue, Storage: "memory", Prefix: "bad:prefix"}
	if _, err := New(logicalErrCfg); err == nil || !strings.Contains(err.Error(), "server: stores:") {
		t.Fatalf("New(logical store error) = %v", err)
	}

	missingCredentialCfg := validLayeredConfig(dir)
	delete(missingCredentialCfg.Stores, "credentials")
	if _, err := New(missingCredentialCfg); err == nil || !strings.Contains(err.Error(), "server: credentials store:") {
		t.Fatalf("New(missing credentials store) = %v", err)
	}

	missingMiniMaxCredentialCfg := validLayeredConfig(dir)
	missingMiniMaxCredentialCfg.MiniMax.CredentialsStore = "missing"
	if _, err := New(missingMiniMaxCredentialCfg); err == nil || !strings.Contains(err.Error(), "server: minimax credentials store:") {
		t.Fatalf("New(missing minimax credentials store) = %v", err)
	}

	missingTenantCfg := validLayeredConfig(dir)
	missingTenantCfg.MiniMax.TenantsStore = "missing"
	if _, err := New(missingTenantCfg); err == nil || !strings.Contains(err.Error(), "server: minimax tenants store:") {
		t.Fatalf("New(missing tenant store) = %v", err)
	}

	missingVoicesCfg := validLayeredConfig(dir)
	missingVoicesCfg.MiniMax.VoicesStore = "missing"
	if _, err := New(missingVoicesCfg); err == nil || !strings.Contains(err.Error(), "server: voices store:") {
		t.Fatalf("New(missing voices store) = %v", err)
	}

	missingWorkspacesCfg := validLayeredConfig(dir)
	missingWorkspacesCfg.Workspaces.Store = "missing"
	if _, err := New(missingWorkspacesCfg); err == nil || !strings.Contains(err.Error(), "server: workspaces store:") {
		t.Fatalf("New(missing workspaces store) = %v", err)
	}

	missingWorkflowsCfg := validLayeredConfig(dir)
	missingWorkflowsCfg.Workflows.Store = "missing"
	if _, err := New(missingWorkflowsCfg); err == nil || !strings.Contains(err.Error(), "server: workflows store:") {
		t.Fatalf("New(missing workflows store) = %v", err)
	}

	missingFirmwareMetadataCfg := validLayeredConfig(dir)
	missingFirmwareMetadataCfg.Depots.MetadataStore = "missing"
	if _, err := New(missingFirmwareMetadataCfg); err == nil || !strings.Contains(err.Error(), "server: firmware metadata store:") {
		t.Fatalf("New(missing firmware metadata store) = %v", err)
	}
}

func TestNewWithPreparedConfig(t *testing.T) {
	dir := t.TempDir()
	adminPublicKey := strings.Repeat("ab", giznet.KeySize)
	adminKey, err := giznet.KeyFromHex(adminPublicKey)
	if err != nil {
		t.Fatalf("KeyFromHex error = %v", err)
	}
	srv, err := New(Config{
		ListenAddr:     ":1234",
		AdminPublicKey: adminKey,
		Stores: map[string]stores.Config{
			"mem": {Kind: stores.KindKeyValue, Backend: "memory"},
			"fw":  {Kind: stores.KindFS, Backend: "filesystem", Dir: filepath.Join(dir, "firmware")},
		},
		Peers: PeersConfig{
			Store: "mem",
		},
		Depots: DepotsConfig{Store: "fw"},
	})
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	if srv.PeerStore == nil {
		t.Fatal("PeerStore is nil")
	}
	if srv.DepotStore == nil {
		t.Fatal("DepotStore is nil")
	}
	if srv.PublicKey().String() == "" {
		t.Fatal("PublicKey should not be empty")
	}
	if srv.AdminPublicKey != adminKey {
		t.Fatalf("AdminPublicKey = %v, want %v", srv.AdminPublicKey, adminKey)
	}
}

func TestConfigValidateRequiresStores(t *testing.T) {
	cfg := Config{}
	if err := cfg.validate(); err == nil {
		t.Fatal("validate should fail without required stores")
	}
}

func TestLoadConfigRejectsInvalidAdminPublicKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("admin-public-key: \"not-hex\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile invalid error = %v", err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig should fail for invalid admin public key")
	}

	if err := os.WriteFile(path, []byte("admin-public-key: \""+strings.Repeat("00", giznet.KeySize)+"\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile zero error = %v", err)
	}
	if _, err := LoadConfig(path); err == nil || !strings.Contains(err.Error(), "zero key") {
		t.Fatalf("LoadConfig zero admin public key err = %v", err)
	}
}

func TestLoadConfigAcceptsTextEncodedAdminPublicKey(t *testing.T) {
	adminKey, err := giznet.KeyFromHex(strings.Repeat("ab", giznet.KeySize))
	if err != nil {
		t.Fatalf("KeyFromHex error = %v", err)
	}
	adminKeyText, err := adminKey.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("admin-public-key: "+string(adminKeyText)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error = %v", err)
	}
	if cfg.AdminPublicKey != adminKey {
		t.Fatalf("AdminPublicKey = %v, want %v", cfg.AdminPublicKey, adminKey)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("LoadConfig should fail for a missing file")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("listen: ["), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig should fail for invalid yaml")
	}
}

func TestMergeFileConfigKeepsRuntimeOverrides(t *testing.T) {
	adminKey, err := giznet.KeyFromHex(strings.Repeat("01", giznet.KeySize))
	if err != nil {
		t.Fatalf("KeyFromHex error = %v", err)
	}
	fileAdminKey, err := giznet.KeyFromHex(strings.Repeat("02", giznet.KeySize))
	if err != nil {
		t.Fatalf("KeyFromHex file error = %v", err)
	}
	runtimeCfg := Config{
		ListenAddr:     ":9999",
		AdminPublicKey: adminKey,
		Storage: map[string]storage.Config{
			"runtime-storage": {Kind: "keyvalue", Backend: "memory"},
		},
		Stores: map[string]stores.Config{
			"runtime": {Kind: "keyvalue", Backend: "memory"},
		},
		Peers: PeersConfig{
			Store: "runtime-peers",
		},
		Credentials: CredentialsConfig{Store: "runtime-credentials"},
		MiniMax: MiniMaxConfig{
			TenantsStore:     "runtime-tenants",
			VoicesStore:      "runtime-voices",
			CredentialsStore: "runtime-credentials",
		},
		Workspaces: WorkspacesConfig{Store: "runtime-workspaces"},
		Workflows:  WorkflowsConfig{Store: "runtime-workflows"},
		Depots:     DepotsConfig{Store: "runtime-depots"},
	}
	fileCfg := ConfigFile{
		ListenAddr:     ":1234",
		AdminPublicKey: fileAdminKey,
		Storage: map[string]storage.Config{
			"file-storage": {Kind: "keyvalue", Backend: "memory"},
		},
		Stores: map[string]stores.Config{
			"file": {Kind: "keyvalue", Backend: "memory"},
		},
		Peers: PeersConfig{
			Store: "file-peers",
		},
		Credentials: CredentialsConfig{Store: "file-credentials"},
		MiniMax: MiniMaxConfig{
			TenantsStore:     "file-tenants",
			VoicesStore:      "file-voices",
			CredentialsStore: "file-credentials",
		},
		Workspaces: WorkspacesConfig{Store: "file-workspaces"},
		Workflows:  WorkflowsConfig{Store: "file-workflows"},
		Depots:     DepotsConfig{Store: "file-depots"},
	}

	merged, err := mergeFileConfig(runtimeCfg, fileCfg)
	if err != nil {
		t.Fatalf("mergeFileConfig error = %v", err)
	}
	if merged.ListenAddr != ":9999" {
		t.Fatalf("ListenAddr = %q", merged.ListenAddr)
	}
	if merged.AdminPublicKey != runtimeCfg.AdminPublicKey {
		t.Fatalf("AdminPublicKey = %v, want %v", merged.AdminPublicKey, runtimeCfg.AdminPublicKey)
	}
	if len(merged.Stores) != 1 || merged.Stores["runtime"].Backend != "memory" {
		t.Fatalf("Stores = %+v", merged.Stores)
	}
	if len(merged.Storage) != 1 || merged.Storage["runtime-storage"].Backend != "memory" {
		t.Fatalf("Storage = %+v", merged.Storage)
	}
	if merged.Peers.Store != "runtime-peers" {
		t.Fatalf("Peers.Store = %q", merged.Peers.Store)
	}
	if merged.Depots.Store != "runtime-depots" {
		t.Fatalf("Depots.Store = %q", merged.Depots.Store)
	}
	if merged.Credentials.Store != "runtime-credentials" {
		t.Fatalf("Credentials.Store = %q", merged.Credentials.Store)
	}
	if merged.MiniMax.TenantsStore != "runtime-tenants" || merged.MiniMax.VoicesStore != "runtime-voices" || merged.MiniMax.CredentialsStore != "runtime-credentials" {
		t.Fatalf("MiniMax = %+v", merged.MiniMax)
	}
	if merged.Workspaces.Store != "runtime-workspaces" {
		t.Fatalf("Workspaces = %+v", merged.Workspaces)
	}
	if merged.Workflows.Store != "runtime-workflows" {
		t.Fatalf("Workflows.Store = %q", merged.Workflows.Store)
	}
}

func TestValidateReportsSpecificMissingFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "missing peers store",
			cfg:  Config{Depots: DepotsConfig{Store: "d"}},
			want: "server: peers.store is required",
		},
		{
			name: "missing depots store",
			cfg:  Config{Peers: PeersConfig{Store: "g"}},
			want: "server: depots.store is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("validate error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateReportsLayeredStorageMissingFields(t *testing.T) {
	base := Config{
		Storage:     map[string]storage.Config{"memory": {Kind: storage.KindKeyValue, Memory: &storage.MemoryConfig{}}},
		Peers:       PeersConfig{Store: "peers"},
		Credentials: CredentialsConfig{Store: "credentials"},
		MiniMax:     MiniMaxConfig{TenantsStore: "minimax-tenants", VoicesStore: "voices", CredentialsStore: "credentials"},
		Workspaces:  WorkspacesConfig{Store: "workspaces"},
		Workflows:   WorkflowsConfig{Store: "workflows"},
		Depots:      DepotsConfig{Store: "firmware", MetadataStore: "firmware-depots"},
	}
	tests := []struct {
		name string
		edit func(*Config)
		want string
	}{
		{"missing credentials", func(c *Config) { c.Credentials.Store = "" }, "server: credentials.store is required"},
		{"missing minimax tenants", func(c *Config) { c.MiniMax.TenantsStore = "" }, "server: minimax.tenants-store is required"},
		{"missing minimax voices", func(c *Config) { c.MiniMax.VoicesStore = "" }, "server: minimax.voices-store is required"},
		{"missing minimax credentials", func(c *Config) { c.MiniMax.CredentialsStore = "" }, "server: minimax.credentials-store is required"},
		{"missing workspaces", func(c *Config) { c.Workspaces.Store = "" }, "server: workspaces.store is required"},
		{"missing workflows", func(c *Config) { c.Workflows.Store = "" }, "server: workflows.store is required"},
		{"missing depot metadata", func(c *Config) { c.Depots.MetadataStore = "" }, "server: depots.metadata-store is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.edit(&cfg)
			err := cfg.validate()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("validate error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestPrepareConfigGeneratesKeyPairAndDefaultListenAddr(t *testing.T) {
	cfg, err := prepareConfig(Config{
		Peers:  PeersConfig{Store: "g"},
		Depots: DepotsConfig{Store: "d"},
	})
	if err != nil {
		t.Fatalf("prepareConfig error = %v", err)
	}
	if cfg.KeyPair == nil {
		t.Fatal("KeyPair should be generated")
	}
	if cfg.ListenAddr != DefaultConfig().ListenAddr {
		t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, DefaultConfig().ListenAddr)
	}
}

func TestNewRejectsUnknownStores(t *testing.T) {
	_, err := New(Config{
		Stores: map[string]stores.Config{
			"bad": {Kind: "keyvalue", Backend: "unknown"},
		},
		Peers:  PeersConfig{Store: "bad"},
		Depots: DepotsConfig{Store: "bad"},
	})
	if err == nil || !strings.Contains(err.Error(), "server: stores:") {
		t.Fatalf("New error = %v", err)
	}
}

func TestNewRejectsMissingNamedStores(t *testing.T) {
	dir := t.TempDir()

	_, err := New(Config{
		Stores: map[string]stores.Config{
			"fw": {Kind: "filestore", Backend: "filesystem", Dir: filepath.Join(dir, "firmware")},
		},
		Peers:  PeersConfig{Store: "missing"},
		Depots: DepotsConfig{Store: "fw"},
	})
	if err == nil || !strings.Contains(err.Error(), "server: peers store:") {
		t.Fatalf("New error = %v", err)
	}

	_, err = New(Config{
		Stores: map[string]stores.Config{
			"mem": {Kind: "keyvalue", Backend: "memory"},
		},
		Peers:  PeersConfig{Store: "mem"},
		Depots: DepotsConfig{Store: "missing"},
	})
	if err == nil || !strings.Contains(err.Error(), "server: firmware store:") {
		t.Fatalf("New error = %v", err)
	}
}

func validLayeredConfig(dir string) Config {
	return Config{
		ListenAddr: ":1234",
		Storage: map[string]storage.Config{
			"memory":      {Kind: storage.KindKeyValue, Memory: &storage.MemoryConfig{}},
			"local-files": {Kind: storage.KindFilesystem, FS: &storage.FSConfig{Dir: dir}},
			"firmware-depot": {
				Kind:    storage.KindDepotStore,
				DepotFS: &storage.DepotFSConfig{},
			},
		},
		Stores: map[string]stores.Config{
			"peers":           {Kind: stores.KindKeyValue, Storage: "memory", Prefix: "peers"},
			"credentials":     {Kind: stores.KindKeyValue, Storage: "memory", Prefix: "credentials"},
			"minimax-tenants": {Kind: stores.KindKeyValue, Storage: "memory", Prefix: "minimax-tenants"},
			"voices":          {Kind: stores.KindKeyValue, Storage: "memory", Prefix: "voices"},
			"workspaces":      {Kind: stores.KindKeyValue, Storage: "memory", Prefix: "workspaces"},
			"workflows":       {Kind: stores.KindKeyValue, Storage: "memory", Prefix: "workflows"},
			"firmware-depots": {Kind: stores.KindKeyValue, Storage: "memory", Prefix: "firmware-depots"},
			"firmware": {
				Kind:    stores.KindDepotStore,
				Storage: "firmware-depot",
				DepotFS: &stores.DepotFSRef{
					Filesystem: storage.FilesystemRef{Storage: "local-files", BaseDir: "firmware"},
				},
			},
		},
		Peers:       PeersConfig{Store: "peers"},
		Credentials: CredentialsConfig{Store: "credentials"},
		MiniMax: MiniMaxConfig{
			TenantsStore:     "minimax-tenants",
			VoicesStore:      "voices",
			CredentialsStore: "credentials",
		},
		Workspaces: WorkspacesConfig{Store: "workspaces"},
		Workflows:  WorkflowsConfig{Store: "workflows"},
		Depots:     DepotsConfig{Store: "firmware", MetadataStore: "firmware-depots"},
	}
}
