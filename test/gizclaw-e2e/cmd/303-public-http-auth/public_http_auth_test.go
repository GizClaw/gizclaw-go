package publichttpauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestPublicHTTPAuthUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "303-public-http-auth")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("device-http").MustSucceed(t)
	serverInfoResp, err := http.Get(h.PublicHTTPURL() + "/api/public/server-info")
	if err != nil {
		t.Fatalf("GET server-info: %v", err)
	}
	if serverInfoResp.StatusCode != http.StatusOK {
		t.Fatalf("GET server-info status = %d", serverInfoResp.StatusCode)
	}
	_ = serverInfoResp.Body.Close()

	reqBody := gearservice.RegistrationRequest{
		Device: apitypes.DeviceInfo{
			Name: strPtr("device-http"),
			Sn:   strPtr("device-http-sn"),
		},
	}
	resp := doJSON(t, http.MethodPost, h.PublicHTTPURL()+"/api/gear/registration", "", reqBody)
	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("unauth register status = %d body=%s", resp.StatusCode, string(body))
	}
	_ = resp.Body.Close()

	session := h.PublicHTTPLogin("device-http")
	resp = doJSON(t, http.MethodPost, h.PublicHTTPURL()+"/api/gear/registration", session.AccessToken, reqBody)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("register status = %d body=%s", resp.StatusCode, string(body))
	}
	_ = resp.Body.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, h.PublicHTTPURL()+"/api/gear/registration", nil)
	if err != nil {
		t.Fatalf("create registration request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET gear registration: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET gear registration status = %d body=%s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), `"role":"unspecified"`) || !strings.Contains(string(body), `"status":"active"`) {
		t.Fatalf("registration did not include active unspecified role:\n%s", string(body))
	}
}

func doJSON(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func strPtr(value string) *string {
	return &value
}
