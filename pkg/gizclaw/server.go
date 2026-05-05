package gizclaw

import (
	"context"
	"errors"
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/credential"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/firmware"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gear"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/mmx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/publiclogin"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/resourcemanager"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/workspace"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/workspacetemplate"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/GizClaw/gizclaw-go/pkg/store/depotstore"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

// Server holds peer transport configuration. Per-stream protocol handling can be
// extended later.
//
// Set gear/firmware storage config on the struct, then call ListenAndServe.
// Internal runtime state is built automatically on first ListenAndServe.
type Server struct {
	KeyPair    *giznet.KeyPair
	ListenAddr string

	GearStore              kv.Store
	CredentialStore        kv.Store
	MiniMaxCredentialStore kv.Store
	MiniMaxTenantStore     kv.Store
	VoiceStore             kv.Store
	WorkspaceStore         kv.Store
	TemplateStore          kv.Store
	WorkspaceTemplateStore kv.Store
	PublicLoginStore       kv.Store
	BuildCommit            string
	ServerPublicKey        string
	AdminPublicKey         string
	DepotStore             depotstore.Store
	DepotMetadataStore     kv.Store

	manager     *Manager
	peerService *PeerService
	sessions    *publiclogin.SessionManager
	listener    *giznet.Listener
}

// Listen initializes the server runtime and binds the UDP peer listener.
func (s *Server) Listen() error {
	if err := s.init(); err != nil {
		return err
	}
	if s.listener != nil {
		return nil
	}
	cfg := giznet.ListenConfig{
		Addr:             s.ListenAddr,
		SecurityPolicy:   (*ServerSecurityPolicy)(s),
		PeerEventHandler: (*serverPeerEventHandler)(s),
	}
	l, err := (&cfg).Listen(s.KeyPair)
	if err != nil {
		return err
	}
	s.listener = l
	return nil
}

// Serve blocks serving accepted peer connections from a listener created by Listen.
func (s *Server) Serve() error {
	l := s.listener
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, giznet.ErrClosed) {
				return nil
			}
			_ = l.Close()
			return err
		}
		svc := s.peerService
		if svc == nil {
			svc = &PeerService{}
		}
		host := &GearConn{
			Conn:    conn,
			Service: svc,
		}
		go func() {
			_ = host.serve()
		}()
	}
}

type serverPeerEventHandler Server

var _ giznet.PeerEventHandler = (*serverPeerEventHandler)(nil)

func (h *serverPeerEventHandler) HandlePeerEvent(ev giznet.PeerEvent) {
	switch ev.State {
	case giznet.PeerStateOffline:
		(*Server)(h).manager.SetPeerDown(ev.PublicKey)
	}
}

// PublicKey returns the configured server public key.
func (s *Server) PublicKey() giznet.PublicKey {
	if s == nil {
		return giznet.PublicKey{}
	}
	if s.KeyPair != nil {
		return s.KeyPair.Public
	}
	return giznet.PublicKey{}
}

// PeerService returns the initialized peer service bundle, or nil before runtime initialization.
func (s *Server) PeerService() *PeerService {
	if s == nil {
		return nil
	}
	return s.peerService
}

// Manager returns the initialized peer manager, or nil before runtime initialization.
func (s *Server) Manager() *Manager {
	if s == nil {
		return nil
	}
	return s.manager
}

func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	listener := s.listener
	var errs []error
	if listener != nil {
		errs = append(errs, listener.Close())
	}
	return errors.Join(errs...)
}

func (s *Server) init() error {
	if s == nil {
		return errors.New("gizclaw: nil server")
	}
	if s.KeyPair == nil {
		return errors.New("gizclaw: nil key pair")
	}
	if s.manager != nil && s.peerService != nil && s.sessions != nil {
		return nil
	}
	if s.GearStore == nil {
		return errors.New("gizclaw: nil gear store")
	}
	if s.DepotStore == nil {
		return errors.New("gizclaw: nil depot store")
	}
	serverPublicKey := s.ServerPublicKey
	if serverPublicKey == "" && s.KeyPair != nil {
		serverPublicKey = s.KeyPair.Public.String()
	}
	if serverPublicKey == "" {
		return fmt.Errorf("gizclaw: empty server public key")
	}

	legacySharedStore := s.CredentialStore == nil &&
		s.MiniMaxTenantStore == nil &&
		s.VoiceStore == nil &&
		s.MiniMaxCredentialStore == nil &&
		s.WorkspaceStore == nil &&
		s.TemplateStore == nil &&
		s.WorkspaceTemplateStore == nil &&
		s.DepotMetadataStore == nil &&
		s.PublicLoginStore == nil
	gearStore := s.GearStore
	if legacySharedStore {
		gearStore = kv.Prefixed(s.GearStore, kv.Key{"gears"})
	}
	credentialStore := moduleStore(s.CredentialStore, s.GearStore, "credentials")
	miniMaxCredentialStore := moduleStore(s.MiniMaxCredentialStore, credentialStore, "")
	miniMaxTenantStore := moduleStore(s.MiniMaxTenantStore, s.GearStore, "minimax-tenants")
	voiceStore := moduleStore(s.VoiceStore, s.GearStore, "voices")
	workspaceStore := moduleStore(s.WorkspaceStore, s.GearStore, "workspaces")
	templateStore := moduleStore(s.TemplateStore, s.GearStore, "workspace-templates")
	workspaceTemplateStore := moduleStore(s.WorkspaceTemplateStore, templateStore, "")
	depotMetadataStore := moduleStore(s.DepotMetadataStore, s.GearStore, "firmware-depots")
	publicLoginStore := moduleStore(s.PublicLoginStore, s.GearStore, "public-login")

	publicLoginServer := publiclogin.NewServer(s.KeyPair, publicLoginStore)
	sessions := publicLoginServer.SessionManager()
	gearsServer := &gear.Server{
		Store:           gearStore,
		BuildCommit:     s.BuildCommit,
		ServerPublicKey: serverPublicKey,
	}
	manager := NewManager(gearsServer)
	gearsServer.PeerManager = manager

	firmwareServer := &firmware.Server{
		Store:         s.DepotStore,
		MetadataStore: depotMetadataStore,
		ResolveGearTarget: func(ctx context.Context, publicKey string) (string, firmware.Channel, error) {
			return resolveGearTarget(ctx, gearsServer, publicKey)
		},
	}
	workspaceTemplateServer := &workspacetemplate.Server{Store: templateStore}
	workspaceServer := &workspace.Server{Store: workspaceStore, TemplateStore: workspaceTemplateStore}
	credentialServer := &credential.Server{Store: credentialStore}
	mmxServer := &mmx.Server{
		TenantStore:     miniMaxTenantStore,
		VoiceStore:      voiceStore,
		CredentialStore: miniMaxCredentialStore,
	}
	resourceManager := resourcemanager.New(resourcemanager.Services{
		Credentials:        credentialServer,
		Gears:              gearsServer,
		MiniMax:            mmxServer,
		Workspaces:         workspaceServer,
		WorkspaceTemplates: workspaceTemplateServer,
	})

	s.manager = manager
	s.peerService = &PeerService{
		manager: manager,
		admin: &adminService{
			CredentialAdminService:        credentialServer,
			FirmwareAdminService:          firmwareServer,
			GearsAdminService:             gearsServer,
			MiniMaxAdminService:           mmxServer,
			WorkspaceAdminService:         workspaceServer,
			WorkspaceTemplateAdminService: workspaceTemplateServer,
			ResourceManager:               resourceManager,
		},
		gear: &gearAPIBundle{
			FirmwareGearService: firmwareServer,
			GearsGearService:    gearsServer,
		},
		public: &serverPublic{
			GearsServerPublic: gearsServer,
			ServerPublic:      publicLoginServer,
		},
	}
	s.sessions = sessions
	return nil
}

func moduleStore(configured, fallback kv.Store, defaultPrefix string) kv.Store {
	if configured != nil {
		return configured
	}
	if fallback == nil {
		return nil
	}
	if defaultPrefix == "" {
		return fallback
	}
	return kv.Prefixed(fallback, kv.Key{defaultPrefix})
}

func resolveGearTarget(ctx context.Context, gearsServer *gear.Server, publicKey string) (string, firmware.Channel, error) {
	if gearsServer == nil {
		return "", "", errors.New("gizclaw: gears service not configured")
	}
	gear, err := gearsServer.LoadGear(ctx, publicKey)
	if err != nil {
		return "", "", err
	}
	if gear.Device.Hardware == nil || gear.Device.Hardware.Depot == nil {
		return "", "", errors.New("missing depot")
	}
	if gear.Configuration.Firmware == nil || gear.Configuration.Firmware.Channel == nil {
		return "", "", errors.New("missing channel")
	}
	channel := firmware.Channel(*gear.Configuration.Firmware.Channel)
	if !channel.Valid() {
		return "", "", fmt.Errorf("invalid firmware channel %q", channel)
	}
	return *gear.Device.Hardware.Depot, channel, nil
}
