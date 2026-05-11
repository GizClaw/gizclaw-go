package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw"
	adminui "github.com/GizClaw/gizclaw-go/cmd/ui/admin"
	playui "github.com/GizClaw/gizclaw-go/cmd/ui/play"
	"github.com/pion/webrtc/v4"
)

const uiAPIProxyTimeout = 30 * time.Second

func ListenAndServeAdminUI(ctxName, addr string, out io.Writer) error {
	return listenAndServeUI(ctxName, addr, "GizClaw Admin UI", adminui.FS(), out, nil, registerAdminUIRoutes)
}

func ListenAndServePlayUI(ctxName, addr string, out io.Writer) error {
	return listenAndServeUI(ctxName, addr, "GizClaw Play UI", playui.FS(), out, ensurePlayRegistration, registerPlayUIRoutes(ctxName))
}

func listenAndServeUI(
	ctxName, addr, title string,
	uiFS fs.FS,
	out io.Writer,
	beforeServe func(context.Context, *gizclaw.Client) error,
	registerRoutes func(*http.ServeMux),
) error {
	if strings.TrimSpace(addr) == "" {
		return fmt.Errorf("gizclaw: empty listen addr")
	}
	listener, err := net.Listen("tcp", normalizeListenAddr(addr))
	if err != nil {
		return fmt.Errorf("gizclaw: listen ui: %w", err)
	}

	c, err := ConnectFromContext(ctxName)
	if err != nil {
		_ = listener.Close()
		return err
	}
	apiProxy := newUIAPIProxy(func() (uiAPIProxyClient, error) {
		return ConnectFromContext(ctxName)
	}, uiAPIProxyTimeout)
	apiProxy.set(c)
	defer apiProxy.Close()

	if beforeServe != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := beforeServe(ctx, c); err != nil {
			_ = listener.Close()
			return err
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiProxy)
	mux.Handle("/api", apiProxy)
	if registerRoutes != nil {
		registerRoutes(mux)
	}
	mux.Handle("/", staticWithSPAFallback(uiFS))

	server := &http.Server{
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	if out != nil {
		_, _ = fmt.Fprintf(out, "%s listening on %s\n", title, displayURL(listener.Addr()))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	err = server.Serve(listener)
	if err == nil || err == http.ErrServerClosed {
		return nil
	}
	return err
}

type uiAPIProxyClient interface {
	Close() error
	ProxyHandler() http.Handler
}

type uiAPIProxy struct {
	connect func() (uiAPIProxyClient, error)
	timeout time.Duration

	mu     sync.Mutex
	client uiAPIProxyClient
}

func newUIAPIProxy(connect func() (uiAPIProxyClient, error), timeout time.Duration) *uiAPIProxy {
	if timeout <= 0 {
		timeout = uiAPIProxyTimeout
	}
	return &uiAPIProxy{
		connect: connect,
		timeout: timeout,
	}
}

func (p *uiAPIProxy) set(client uiAPIProxyClient) {
	p.mu.Lock()
	p.client = client
	p.mu.Unlock()
}

func (p *uiAPIProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response, client, err := p.serveOnce(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if isUIAPIProxyRetryable(r) && isUIAPIProxyFailure(response.statusCode()) {
		p.invalidate(client)
		response, _, err = p.serveOnce(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	response.writeTo(w)
}

func (p *uiAPIProxy) serveOnce(r *http.Request) (*bufferedHTTPResponse, uiAPIProxyClient, error) {
	client, err := p.get()
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(r.Context(), p.timeout)
	defer cancel()

	response := newBufferedHTTPResponse()
	client.ProxyHandler().ServeHTTP(response, r.WithContext(ctx))
	if ctx.Err() != nil {
		p.invalidate(client)
	}
	return response, client, nil
}

func (p *uiAPIProxy) Close() error {
	p.mu.Lock()
	client := p.client
	p.client = nil
	p.mu.Unlock()
	if client != nil {
		return client.Close()
	}
	return nil
}

func (p *uiAPIProxy) get() (uiAPIProxyClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		return p.client, nil
	}
	client, err := p.connect()
	if err != nil {
		return nil, err
	}
	p.client = client
	return client, nil
}

func (p *uiAPIProxy) invalidate(stale uiAPIProxyClient) {
	p.mu.Lock()
	if p.client != stale {
		p.mu.Unlock()
		return
	}
	p.client = nil
	p.mu.Unlock()
	_ = stale.Close()
}

func isUIAPIProxyRetryable(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func isUIAPIProxyFailure(statusCode int) bool {
	switch statusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

type bufferedHTTPResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedHTTPResponse() *bufferedHTTPResponse {
	return &bufferedHTTPResponse{header: make(http.Header)}
}

func (r *bufferedHTTPResponse) Header() http.Header {
	return r.header
}

func (r *bufferedHTTPResponse) WriteHeader(statusCode int) {
	if r.status != 0 {
		return
	}
	r.status = statusCode
}

func (r *bufferedHTTPResponse) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *bufferedHTTPResponse) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *bufferedHTTPResponse) writeTo(w http.ResponseWriter) {
	for key, values := range r.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(r.statusCode())
	_, _ = r.body.WriteTo(w)
}

func ensurePlayRegistration(ctx context.Context, c *gizclaw.Client) error {
	_, err := c.GetGearRegistration(ctx, "gear.registration.get")
	return err
}

type playWebRTCOfferRequest struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
}

type playWebRTCAnswerResponse struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
}

func registerPlayUIRoutes(ctxName string) func(*http.ServeMux) {
	return func(mux *http.ServeMux) {
		mux.HandleFunc("/webrtc/offer", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", http.MethodPost)
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}
			handlePlayWebRTCOffer(w, r, ctxName)
		})
	}
}

func registerAdminUIRoutes(mux *http.ServeMux) {
	redirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ai/workflows", http.StatusFound)
	}
	mux.HandleFunc("/workspace-templates", redirect)
	mux.HandleFunc("/workspace-templates/", redirect)
	mux.HandleFunc("/ai/workspace-templates", redirect)
	mux.HandleFunc("/ai/workspace-templates/", redirect)
}

func handlePlayWebRTCOffer(w http.ResponseWriter, r *http.Request, ctxName string) {
	var req playWebRTCOfferRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid offer json", http.StatusBadRequest)
		return
	}
	if req.Type != webrtc.SDPTypeOffer.String() || strings.TrimSpace(req.SDP) == "" {
		http.Error(w, "invalid webrtc offer", http.StatusBadRequest)
		return
	}
	c, err := ConnectFromContext(ctxName)
	if err != nil {
		playWebRTCError(w, "connect client failed", err, http.StatusServiceUnavailable)
		return
	}
	var closeClientOnce sync.Once
	closeClient := func() {
		closeClientOnce.Do(func() {
			_ = c.Close()
		})
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		closeClient()
		playWebRTCError(w, "create peer connection failed", err, http.StatusInternalServerError)
		return
	}
	registration, err := c.RegisterTo(pc)
	if err != nil {
		_ = pc.Close()
		closeClient()
		playWebRTCError(w, "register peer connection failed", err, http.StatusInternalServerError)
		return
	}
	closeWebRTC := func() {
		_ = registration.Close()
		_ = pc.Close()
		closeClient()
	}
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateClosed:
			closeWebRTC()
		}
	})

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: req.SDP}); err != nil {
		closeWebRTC()
		playWebRTCError(w, "set remote description failed", err, http.StatusBadRequest)
		return
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		closeWebRTC()
		playWebRTCError(w, "create answer failed", err, http.StatusInternalServerError)
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		closeWebRTC()
		playWebRTCError(w, "set local description failed", err, http.StatusInternalServerError)
		return
	}
	select {
	case <-gatherComplete:
	case <-r.Context().Done():
		closeWebRTC()
		return
	}

	local := pc.LocalDescription()
	if local == nil {
		closeWebRTC()
		playWebRTCError(w, "missing local description", fmt.Errorf("local description is nil"), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(playWebRTCAnswerResponse{SDP: local.SDP, Type: local.Type.String()}); err != nil {
		closeWebRTC()
	}
}

func playWebRTCError(w http.ResponseWriter, message string, err error, status int) {
	slog.Error("gizclaw: play webrtc signaling failed", "message", message, "error", err, "status", status)
	http.Error(w, fmt.Sprintf("%s: %v", message, err), status)
}

// staticWithSPAFallback serves embedded UI assets and falls back to index.html
// for client-side routes (e.g. /peers/...) so BrowserRouter deep links work.
func staticWithSPAFallback(uiFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(uiFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean != "" {
			if _, err := fs.Stat(uiFS, clean); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r2 := r.Clone(r.Context())
		r2.URL = r.URL
		u := *r.URL
		u.Path = "/"
		r2.URL = &u
		fileServer.ServeHTTP(w, r2)
	})
}

func normalizeListenAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return addr
	}
	if strings.Contains(addr, ":") {
		return addr
	}
	return ":" + addr
}

func displayURL(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String()
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}
