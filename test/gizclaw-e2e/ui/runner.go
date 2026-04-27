package ui_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	adminui "github.com/GizClaw/gizclaw-go/ui/apps/admin"
	playui "github.com/GizClaw/gizclaw-go/ui/apps/play"
	"github.com/goccy/go-yaml"
	"github.com/playwright-community/playwright-go"

	itest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/testutil"
)

const (
	SeedDepotName             = itest.SeedDepotName
	SeedCredentialName        = itest.SeedCredentialName
	SeedMiniMaxTenantName     = itest.SeedMiniMaxTenantName
	SeedVoiceID               = itest.SeedVoiceID
	SeedWorkspaceTemplateName = itest.SeedWorkspaceTemplateName
	SeedWorkspaceName         = itest.SeedWorkspaceName
)

type Story struct {
	Name string
	Run  func(testing.TB, *Page)
}

type Seed struct {
	AdminURL              string
	PlayURL               string
	ErrorPlayURL          string
	DevicePublicKey       string
	ActionDevicePublicKey string
	DeleteDevicePublicKey string
}

type Page struct {
	t    testing.TB
	page playwright.Page
	Seed Seed
}

type Suite struct {
	t       testing.TB
	seed    Seed
	runner  *browserRunner
	context playwright.BrowserContext
}

type browserRunner struct {
	browser playwright.Browser
	pw      *playwright.Playwright
}

type cliContext struct {
	Name      string
	KeyPair   *giznet.KeyPair
	Server    serverConfig
	ServerKey giznet.PublicKey
}

type serverConfig struct {
	Address   string `yaml:"address"`
	PublicKey string `yaml:"public-key"`
}

type contextConfig struct {
	Server serverConfig `yaml:"server"`
}

func RunStories(t *testing.T, stories []Story) {
	t.Helper()

	suite := NewSuite(t)
	defer suite.Close()

	for _, story := range stories {
		story := story
		t.Run(story.Name, func(t *testing.T) {
			suite.RunStory(t, story.Run)
		})
	}
}

func NewSuite(t testing.TB) *Suite {
	t.Helper()

	seed := startSeededUI(t)
	runner := newBrowserRunner(t)
	ctx, err := runner.browser.NewContext()
	if err != nil {
		runner.close(t)
		t.Fatalf("create browser context: %v", err)
	}
	return &Suite{t: t, seed: seed, runner: runner, context: ctx}
}

func (s *Suite) RunStory(t testing.TB, run func(testing.TB, *Page)) {
	t.Helper()

	page, err := s.context.NewPage()
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	defer page.Close()

	run(t, &Page{t: t, page: page, Seed: s.seed})
}

func (s *Suite) Close() {
	s.t.Helper()
	if err := s.context.Close(); err != nil {
		s.t.Fatalf("close browser context: %v", err)
	}
	s.runner.close(s.t)
}

func (p *Page) GotoAdmin(routePath string) {
	p.gotoURL(p.Seed.AdminURL, routePath)
}

func (p *Page) GotoPlay(routePath string) {
	p.gotoURL(p.Seed.PlayURL, routePath)
}

func (p *Page) GotoErrorPlay(routePath string) {
	p.gotoURL(p.Seed.ErrorPlayURL, routePath)
}

func (p *Page) ExpectURLSuffix(suffix string) {
	p.t.Helper()
	if err := itest.WaitUntil(10*time.Second, func() error {
		current := p.page.URL()
		if strings.HasSuffix(current, suffix) {
			return nil
		}
		return fmt.Errorf("url %q does not end with %q", current, suffix)
	}); err != nil {
		p.t.Fatal(err)
	}
}

func (p *Page) ExpectText(text string) {
	p.t.Helper()
	if err := itest.WaitUntil(10*time.Second, func() error {
		body, err := p.page.TextContent("body")
		if err != nil {
			return err
		}
		if strings.Contains(body, text) {
			return nil
		}
		return fmt.Errorf("page body does not contain %q; body=%q", text, body)
	}); err != nil {
		p.t.Fatal(err)
	}
}

func (p *Page) Fill(selector, value string) {
	p.t.Helper()
	if err := p.page.Locator(selector).Fill(value); err != nil {
		p.t.Fatalf("fill %q: %v", selector, err)
	}
}

func (p *Page) FillNth(selector string, index int, value string) {
	p.t.Helper()
	if err := p.page.Locator(selector).Nth(index).Fill(value); err != nil {
		p.t.Fatalf("fill %q nth=%d: %v", selector, index, err)
	}
}

func (p *Page) ClickRole(role, name string) {
	p.t.Helper()
	if err := p.page.GetByRole(playwright.AriaRole(role), playwright.PageGetByRoleOptions{
		Name:  name,
		Exact: playwright.Bool(true),
	}).Click(); err != nil {
		p.t.Fatalf("click role=%s name=%q: %v", role, name, err)
	}
}

func (p *Page) ClickNavigationLink(name string) {
	p.t.Helper()
	err := p.page.GetByRole(playwright.AriaRole("navigation")).GetByRole(playwright.AriaRole("link"), playwright.LocatorGetByRoleOptions{
		Name:  name,
		Exact: playwright.Bool(true),
	}).Click()
	if err != nil {
		p.t.Fatalf("click navigation link %q: %v", name, err)
	}
}

func (p *Page) SetInputFiles(index int, name, mimeType string, data []byte) {
	p.t.Helper()
	err := p.page.Locator(`input[type="file"]`).Nth(index).SetInputFiles([]playwright.InputFile{{
		Name:     name,
		MimeType: mimeType,
		Buffer:   data,
	}})
	if err != nil {
		p.t.Fatalf("set input file %d: %v", index, err)
	}
}

func FirmwareReleaseTar(t testing.TB, channel, firmwareSemver string) []byte {
	t.Helper()

	releaseTar, err := itest.FirmwareReleaseTarSeed(channel, firmwareSemver)
	if err != nil {
		t.Fatalf("build firmware tar seed: %v", err)
	}
	return releaseTar
}

func DepotInfoJSON(t testing.TB) []byte {
	t.Helper()

	info, err := itest.LoadDepotInfoSeed()
	if err != nil {
		t.Fatalf("load depot info seed: %v", err)
	}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal depot info seed: %v", err)
	}
	return data
}

func (p *Page) gotoURL(baseURL, routePath string) {
	p.t.Helper()
	target := joinURL(p.t, baseURL, routePath)
	if _, err := p.page.Goto(target); err != nil {
		p.t.Fatalf("goto %s: %v", target, err)
	}
}

func startSeededUI(t testing.TB) Seed {
	t.Helper()

	adminCtx := loadCurrentCLIContext(t)
	adminClient := startExternalClient(t, adminCtx)
	t.Cleanup(func() { _ = adminClient.Close() })

	adminSeed, err := itest.LoadRegistrationSeed("admin")
	if err != nil {
		t.Fatalf("load admin registration seed: %v", err)
	}
	registerGear(t, adminClient, itest.RegistrationRequest(adminCtx.KeyPair.Public.String(), adminSeed))

	adminAPI, err := adminClient.ServerAdminClient()
	if err != nil {
		t.Fatalf("create admin API client: %v; current gizclaw context must be an admin for the running service", err)
	}

	workspaceRoot := filepath.Join(mustRepoRoot(t), "test", "gizclaw-e2e", ".workspace", "ui-real-service")
	deviceCtx := loadOrCreateExternalContext(t, workspaceRoot, "device-a", adminCtx.Server)
	actionDeviceCtx := loadOrCreateExternalContext(t, workspaceRoot, "device-actions-a", adminCtx.Server)
	deleteDeviceCtx := loadOrCreateExternalContext(t, workspaceRoot, "device-delete-a", adminCtx.Server)

	deviceClient := startExternalClient(t, deviceCtx)
	t.Cleanup(func() { _ = deviceClient.Close() })
	actionDeviceClient := startExternalClient(t, actionDeviceCtx)
	t.Cleanup(func() { _ = actionDeviceClient.Close() })
	deleteDeviceClient := startExternalClient(t, deleteDeviceCtx)
	t.Cleanup(func() { _ = deleteDeviceClient.Close() })

	deviceSeed, err := itest.LoadRegistrationSeed("device")
	if err != nil {
		t.Fatalf("load device registration seed: %v", err)
	}
	registerGear(t, deviceClient, itest.RegistrationRequest(deviceCtx.KeyPair.Public.String(), deviceSeed))
	registerGear(t, actionDeviceClient, itest.RegistrationRequest(actionDeviceCtx.KeyPair.Public.String(), deviceSeed))
	registerGear(t, deleteDeviceClient, itest.RegistrationRequest(deleteDeviceCtx.KeyPair.Public.String(), deviceSeed))

	seedCtx, cancel := context.WithTimeout(context.Background(), itest.ReadyTimeout)
	defer cancel()
	if err := itest.ApplyAdminCatalogSeed(seedCtx, adminAPI); err != nil {
		t.Fatalf("apply admin catalog seed: %v", err)
	}
	if err := itest.ApplyWorkspaceSeed(seedCtx, adminAPI); err != nil {
		t.Fatalf("apply workspace seed: %v", err)
	}
	if err := itest.ApplyFirmwareSeed(seedCtx, adminAPI); err != nil {
		t.Fatalf("apply firmware seed: %v", err)
	}
	applyFirmwareReleaseSeed(t, seedCtx, adminAPI, "beta", "1.0.1")
	for _, publicKey := range []string{deviceCtx.KeyPair.Public.String(), actionDeviceCtx.KeyPair.Public.String(), deleteDeviceCtx.KeyPair.Public.String()} {
		approveGear(t, seedCtx, adminAPI, publicKey)
		if err := itest.ApplyDeviceConfigSeed(seedCtx, adminAPI, publicKey); err != nil {
			t.Fatalf("apply device config seed for %q: %v", publicKey, err)
		}
	}

	return Seed{
		AdminURL:              startTestUI(t, "admin", adminClient, adminui.FS()),
		PlayURL:               startTestUI(t, "play", deviceClient, playui.FS()),
		ErrorPlayURL:          startErrorTestUI(t, "play-error", playui.FS()),
		DevicePublicKey:       deviceCtx.KeyPair.Public.String(),
		ActionDevicePublicKey: actionDeviceCtx.KeyPair.Public.String(),
		DeleteDevicePublicKey: deleteDeviceCtx.KeyPair.Public.String(),
	}
}

func registerGear(t testing.TB, client *gizclaw.Client, request serverpublic.RegistrationRequest) {
	t.Helper()

	api, err := client.ServerPublicClient()
	if err != nil {
		t.Fatalf("create public API client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), itest.ReadyTimeout)
	defer cancel()
	resp, err := api.RegisterGearWithResponse(ctx, request)
	if err != nil {
		t.Fatalf("register %q: %v", request.PublicKey, err)
	}
	if resp.JSON200 != nil || resp.StatusCode() == http.StatusConflict {
		return
	}
	t.Fatalf("register %q got status %d: %s", request.PublicKey, resp.StatusCode(), strings.TrimSpace(string(resp.Body)))
}

func approveGear(t testing.TB, ctx context.Context, api *adminservice.ClientWithResponses, publicKey string) {
	t.Helper()

	resp, err := api.ApproveGearWithResponse(ctx, publicKey, adminservice.ApproveRequest{Role: apitypes.GearRoleDevice})
	if err != nil {
		t.Fatalf("approve %q: %v", publicKey, err)
	}
	if resp.JSON200 != nil {
		return
	}
	t.Fatalf("approve %q got status %d: %s", publicKey, resp.StatusCode(), strings.TrimSpace(string(resp.Body)))
}

func applyFirmwareReleaseSeed(t testing.TB, ctx context.Context, api *adminservice.ClientWithResponses, channel, firmwareSemver string) {
	t.Helper()

	releaseTar, err := itest.FirmwareReleaseTarSeed(channel, firmwareSemver)
	if err != nil {
		t.Fatalf("build %s firmware seed: %v", channel, err)
	}
	resp, err := api.PutChannelWithBodyWithResponse(ctx, SeedDepotName, channel, "application/octet-stream", bytes.NewReader(releaseTar))
	if err != nil {
		t.Fatalf("put %s firmware seed: %v", channel, err)
	}
	if resp.JSON200 == nil {
		t.Fatalf("put %s firmware seed got status %d: %s", channel, resp.StatusCode(), strings.TrimSpace(string(resp.Body)))
	}
}

func startTestUI(t testing.TB, name string, client *gizclaw.Client, uiFS fs.FS) string {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("/api/", client.ProxyHandler())
	mux.Handle("/api", client.ProxyHandler())
	mux.Handle("/", staticWithSPAFallback(uiFS))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	t.Logf("%s test UI listening on %s", name, server.URL)
	return server.URL
}

func startErrorTestUI(t testing.TB, name string, uiFS fs.FS) string {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no gizclaw client configured for error scenario", http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no gizclaw client configured for error scenario", http.StatusServiceUnavailable)
	})
	mux.Handle("/", staticWithSPAFallback(uiFS))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	t.Logf("%s test UI listening on %s", name, server.URL)
	return server.URL
}

func staticWithSPAFallback(uiFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(uiFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean != "" {
			info, err := fs.Stat(uiFS, clean)
			if err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		index, err := uiFS.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		defer index.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, index)
	})
}

func newBrowserRunner(t testing.TB) *browserRunner {
	t.Helper()

	options := &playwright.RunOptions{
		Browsers:         []string{"chromium"},
		OnlyInstallShell: true,
		Stdout:           io.Discard,
		Stderr:           io.Discard,
	}
	pw, err := playwright.Run(options)
	if err != nil {
		t.Fatalf("start Playwright: %v\nInstall Playwright for Go explicitly before running UI tests.", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		_ = pw.Stop()
		t.Fatalf("launch Chromium: %v", err)
	}
	return &browserRunner{browser: browser, pw: pw}
}

func (r *browserRunner) close(t testing.TB) {
	t.Helper()
	if err := r.browser.Close(); err != nil {
		t.Fatalf("close browser: %v", err)
	}
	if err := r.pw.Stop(); err != nil {
		t.Fatalf("stop Playwright: %v", err)
	}
}

func startExternalClient(t testing.TB, cliCtx cliContext) *gizclaw.Client {
	t.Helper()

	client := &gizclaw.Client{KeyPair: cliCtx.KeyPair}
	errCh := make(chan error, 1)
	go func() {
		errCh <- client.DialAndServe(cliCtx.ServerKey, cliCtx.Server.Address)
	}()

	if err := itest.WaitUntil(5*time.Second, func() error {
		select {
		case err := <-errCh:
			return fmt.Errorf("connect to running gizclaw service at %s: %w", cliCtx.Server.Address, err)
		default:
		}
		if client.PeerConn() != nil {
			return nil
		}
		return fmt.Errorf("waiting for gizclaw service connection at %s", cliCtx.Server.Address)
	}); err != nil {
		t.Fatalf("%v\nStart the service before running UI e2e tests.", err)
	}
	return client
}

func loadCurrentCLIContext(t testing.TB) cliContext {
	t.Helper()

	root := configDir(t)
	link := filepath.Join(root, "current")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("load current gizclaw context: %v; run 'gizclaw context create' and 'gizclaw service start' first", err)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	return loadCLIContext(t, target)
}

func loadOrCreateExternalContext(t testing.TB, workspaceRoot, name string, server serverConfig) cliContext {
	t.Helper()

	dir := filepath.Join(workspaceRoot, "contexts", name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("create UI e2e context dir: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		data, err := yaml.Marshal(contextConfig{Server: server})
		if err != nil {
			t.Fatalf("marshal UI e2e context config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0o644); err != nil {
			t.Fatalf("write UI e2e context config: %v", err)
		}
	} else if err != nil {
		t.Fatalf("stat UI e2e context config: %v", err)
	}

	keyPath := filepath.Join(dir, "identity.key")
	if _, err := loadOrGenerateKeyPair(keyPath); err != nil {
		t.Fatalf("load UI e2e identity %q: %v", name, err)
	}
	return loadCLIContext(t, dir)
}

func loadCLIContext(t testing.TB, dir string) cliContext {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("read gizclaw context config %q: %v", dir, err)
	}
	var cfg contextConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse gizclaw context config %q: %v", dir, err)
	}
	keyPair, err := loadOrGenerateKeyPair(filepath.Join(dir, "identity.key"))
	if err != nil {
		t.Fatalf("load gizclaw context identity %q: %v", dir, err)
	}
	serverKey, err := giznet.KeyFromHex(cfg.Server.PublicKey)
	if err != nil {
		t.Fatalf("parse gizclaw server public key: %v", err)
	}
	return cliContext{
		Name:      filepath.Base(filepath.Clean(dir)),
		KeyPair:   keyPair,
		Server:    cfg.Server,
		ServerKey: serverKey,
	}
}

func loadOrGenerateKeyPair(keyPath string) (*giznet.KeyPair, error) {
	data, err := os.ReadFile(keyPath)
	if err == nil {
		if len(data) != giznet.KeySize {
			return nil, fmt.Errorf("invalid key size: got %d, want %d", len(data), giznet.KeySize)
		}
		var key giznet.Key
		copy(key[:], data)
		return giznet.NewKeyPair(key)
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	var key giznet.Key
	if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
		return nil, err
	}
	keyPair, err := giznet.NewKeyPair(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, keyPair.Private[:], 0o600); err != nil {
		return nil, err
	}
	return keyPair, nil
}

func configDir(t testing.TB) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "gizclaw")
		}
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("resolve home dir: %v", err)
		}
		return filepath.Join(home, ".config", "gizclaw")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("resolve user config dir: %v", err)
	}
	return filepath.Join(dir, "gizclaw")
}

func joinURL(t testing.TB, baseURL, routePath string) string {
	t.Helper()

	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base URL %q: %v", baseURL, err)
	}
	parsed.Path = routePath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func mustRepoRoot(t testing.TB) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve UI test helper path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolve repo root %q: %v", root, err)
	}
	return root
}
