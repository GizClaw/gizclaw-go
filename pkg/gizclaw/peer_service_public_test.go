package gizclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/publiclogin"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/GizClaw/gizclaw-go/pkg/store/depotstore"
)

func TestPublicHTTPLoginRegisterAndGearAPI(t *testing.T) {
	serverKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(server) error = %v", err)
	}
	deviceKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(device) error = %v", err)
	}
	server := &Server{
		KeyPair:            serverKey,
		GearStore:          mustBadgerInMemory(t, nil),
		DepotStore:         depotstore.Dir(t.TempDir()),
		BuildCommit:        "test-build",
		ServerPublicKey:    serverKey.Public.String(),
		DepotMetadataStore: mustBadgerInMemory(t, nil),
	}
	handler, err := server.PublicHTTPHandler()
	if err != nil {
		t.Fatalf("PublicHTTPHandler error = %v", err)
	}
	ts := httptest.NewServer(handler)
	defer ts.Close()

	infoResp, err := http.Get(ts.URL + "/api/public/server-info")
	if err != nil {
		t.Fatalf("GET server-info error = %v", err)
	}
	if infoResp.StatusCode != http.StatusOK {
		t.Fatalf("GET server-info status = %d", infoResp.StatusCode)
	}
	_ = infoResp.Body.Close()

	reqBody := gearservice.RegistrationRequest{
		Device: apitypes.DeviceInfo{Name: strPtr("device-http")},
	}
	registerResp := doJSON(t, http.MethodPost, ts.URL+"/api/gear/registration", "", reqBody)
	if registerResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth register status = %d", registerResp.StatusCode)
	}
	_ = registerResp.Body.Close()

	session := publicHTTPTestLogin(t, ts.URL, serverKey.Public, deviceKey)
	registerResp = doJSON(t, http.MethodPost, ts.URL+"/api/gear/registration", session.AccessToken, reqBody)
	if registerResp.StatusCode != http.StatusOK {
		t.Fatalf("register status = %d", registerResp.StatusCode)
	}
	_ = registerResp.Body.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/api/gear/registration", nil)
	if err != nil {
		t.Fatalf("NewRequest registration error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET registration error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET registration status = %d", resp.StatusCode)
	}

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/api/gear/registration", nil)
	if err != nil {
		t.Fatalf("NewRequest registration mismatch error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set(publicKeyHeader, serverKey.Public.String())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET registration mismatch error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET registration mismatch status = %d", resp.StatusCode)
	}
}

func publicHTTPTestLogin(t *testing.T, baseURL string, serverPublicKey giznet.PublicKey, deviceKey *giznet.KeyPair) publiclogin.LoginResponse {
	t.Helper()
	assertion, err := publiclogin.NewLoginAssertion(deviceKey, serverPublicKey, time.Minute)
	if err != nil {
		t.Fatalf("NewLoginAssertion error = %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/api/public/login", nil)
	if err != nil {
		t.Fatalf("NewRequest login error = %v", err)
	}
	req.Header.Set(publicKeyHeader, deviceKey.Public.String())
	req.Header.Set("Authorization", "Bearer "+assertion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST login error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST login status = %d", resp.StatusCode)
	}
	var result publiclogin.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return result
}

func doJSON(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewRequest error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s error = %v", method, url, err)
	}
	return resp
}

func strPtr(value string) *string {
	return &value
}
