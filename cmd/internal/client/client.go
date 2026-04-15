package client

import (
	"fmt"

	"github.com/giztoy/giztoy-go/cmd/internal/clicontext"
	"github.com/giztoy/giztoy-go/pkg/gizclaw"
	"github.com/giztoy/giztoy-go/pkg/giznet"
)

func DialFromContext(name string) (*gizclaw.Client, giznet.PublicKey, string, error) {
	store, err := clicontext.DefaultStore()
	if err != nil {
		return nil, giznet.PublicKey{}, "", err
	}
	var cliCtx *clicontext.CLIContext
	if name != "" {
		cliCtx, err = store.LoadByName(name)
	} else {
		cliCtx, err = store.Current()
	}
	if err != nil {
		return nil, giznet.PublicKey{}, "", err
	}
	if cliCtx == nil {
		return nil, giznet.PublicKey{}, "", fmt.Errorf("no active context; run 'giztoy context create' first")
	}
	serverPK, err := cliCtx.ServerPublicKey()
	if err != nil {
		return nil, giznet.PublicKey{}, "", fmt.Errorf("invalid server public key: %w", err)
	}
	return &gizclaw.Client{
		KeyPair: cliCtx.KeyPair,
	}, serverPK, cliCtx.Config.Server.Address, nil
}
