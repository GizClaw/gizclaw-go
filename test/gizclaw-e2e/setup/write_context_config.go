package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

type contextConfig struct {
	Server contextServerConfig `yaml:"server"`
}

type contextServerConfig struct {
	Address    string `yaml:"address"`
	PublicKey  string `yaml:"public-key"`
	CipherMode string `yaml:"cipher-mode"`
}

func main() {
	var contextHome, serverWorkspace, serverAddr, cipherMode, contextName, clientPrivate, wantClientPublic string
	flag.StringVar(&contextHome, "context-home", "", "XDG config home for generated e2e context")
	flag.StringVar(&serverWorkspace, "server-workspace", "", "server workspace directory")
	flag.StringVar(&serverAddr, "server-addr", "", "server listen address")
	flag.StringVar(&cipherMode, "cipher-mode", "chacha_poly", "giznet cipher mode")
	flag.StringVar(&contextName, "context-name", "e2e-client", "client context name")
	flag.StringVar(&clientPrivate, "client-private-key", "", "client private key text")
	flag.StringVar(&wantClientPublic, "client-public-key", "", "optional expected client public key")
	flag.Parse()

	if contextHome == "" {
		fatalf("context-home is required")
	}
	if serverWorkspace == "" {
		fatalf("server-workspace is required")
	}
	if serverAddr == "" {
		fatalf("server-addr is required")
	}
	if clientPrivate == "" {
		fatalf("client-private-key is required")
	}

	serverKP, err := readRawKeyPair(filepath.Join(serverWorkspace, "identity.key"))
	if err != nil {
		fatalf("read server identity: %v", err)
	}
	var clientKey giznet.Key
	if err := clientKey.UnmarshalText([]byte(clientPrivate)); err != nil {
		fatalf("parse client private key: %v", err)
	}
	clientKP, err := giznet.NewKeyPair(clientKey)
	if err != nil {
		fatalf("derive client keypair: %v", err)
	}
	if wantClientPublic != "" && clientKP.Public.String() != wantClientPublic {
		fatalf("client public key mismatch: derived %s, env %s", clientKP.Public.String(), wantClientPublic)
	}

	contextDir := filepath.Join(contextHome, "gizclaw", contextName)
	if err := os.MkdirAll(contextDir, 0o700); err != nil {
		fatalf("create context dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contextDir, "identity.key"), clientKP.Private[:], 0o600); err != nil {
		fatalf("write client identity: %v", err)
	}
	cfg := contextConfig{
		Server: contextServerConfig{
			Address:    serverAddr,
			PublicKey:  serverKP.Public.String(),
			CipherMode: cipherMode,
		},
	}
	data := []byte(fmt.Sprintf("server:\n  address: %s\n  public-key: %s\n  cipher-mode: %s\n", cfg.Server.Address, cfg.Server.PublicKey, cfg.Server.CipherMode))
	if err := os.WriteFile(filepath.Join(contextDir, "config.yaml"), data, 0o644); err != nil {
		fatalf("write context config: %v", err)
	}
	_ = os.Remove(filepath.Join(contextHome, "gizclaw", "current"))
	_ = os.Symlink(contextName, filepath.Join(contextHome, "gizclaw", "current"))
}

func readRawKeyPair(path string) (*giznet.KeyPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		keyPair, err := giznet.GenerateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("generate key pair: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create identity dir: %w", err)
		}
		if err := os.WriteFile(path, keyPair.Private[:], 0o600); err != nil {
			return nil, fmt.Errorf("write generated identity: %w", err)
		}
		return keyPair, nil
	}
	if len(data) != giznet.KeySize {
		return nil, fmt.Errorf("invalid key length: got %d, want %d", len(data), giznet.KeySize)
	}
	var key giznet.Key
	copy(key[:], data)
	return giznet.NewKeyPair(key)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "write_context_config: "+format+"\n", args...)
	os.Exit(1)
}
