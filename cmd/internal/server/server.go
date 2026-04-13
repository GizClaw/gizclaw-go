package server

import (
	"fmt"
	"os"

	"github.com/giztoy/giztoy-go/cmd/internal/stores"
	"github.com/giztoy/giztoy-go/pkg/firmware"
	"github.com/giztoy/giztoy-go/pkg/gears"
	"github.com/giztoy/giztoy-go/pkg/gizclaw"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/adminservice"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/gearservice"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/rpc"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/serverpublic"
)

var BuildCommit = "dev"

// New wires an already prepared in-memory config into a gizclaw.Server.
func New(cfg Config) (*gizclaw.Server, error) {
	cfg, err := prepareConfig(cfg)
	if err != nil {
		return nil, err
	}
	ss, err := stores.New(cfg.Stores)
	if err != nil {
		return nil, fmt.Errorf("server: stores: %w", err)
	}
	closeStores := func() error { return ss.Close() }

	gearsKV, err := ss.KV(cfg.Gears.Store)
	if err != nil {
		_ = ss.Close()
		return nil, fmt.Errorf("server: gears store: %w", err)
	}
	gearStore := gears.NewStore(gearsKV)
	gearService := gears.NewService(gearStore, cfg.Gears.RegistrationTokens)

	fwStore, err := ss.FS(cfg.Depots.Store)
	if err != nil {
		_ = ss.Close()
		return nil, fmt.Errorf("server: firmware store: %w", err)
	}
	if err := os.MkdirAll(fwStore.Root(), 0o755); err != nil {
		_ = ss.Close()
		return nil, fmt.Errorf("server: firmware dir: %w", err)
	}
	fwScanner := firmware.NewScanner(fwStore)
	fwUploader := firmware.NewUploader(fwStore, fwScanner)
	fwSwitcher := firmware.NewSwitcher(fwStore, fwScanner)
	fwOTA := firmware.NewOTAService(fwStore, fwScanner)

	manager := gizclaw.NewManager(gearService)
	peerServer := &gizclaw.PeerServer{
		Manager: manager,
		Admin: &adminservice.Server{
			FirmwareScanner:  fwScanner,
			FirmwareUploader: fwUploader,
			FirmwareSwitcher: fwSwitcher,
		},
		Gear: &gearservice.GearServer{
			Gears:       gearService,
			FirmwareOTA: fwOTA,
			Manager:     manager,
		},
		Public: &serverpublic.PublicServer{
			BuildCommit:     BuildCommit,
			ServerPublicKey: cfg.KeyPair.Public.String(),
			Gears:           gearService,
			FirmwareOTA:     fwOTA,
			PeerServer:      manager,
		},
		RPC: rpc.NewServer(&rpc.RPCServer{}),
	}

	return &gizclaw.Server{
		KeyPair:        cfg.KeyPair,
		Manager:        manager,
		PeerServer:     peerServer,
		SecurityPolicy: gizclaw.GearsSecurityPolicy{Gears: gearService},
		Cleanup:        closeStores,
	}, nil
}
