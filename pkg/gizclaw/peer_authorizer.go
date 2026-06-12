package gizclaw

import (
	"context"
	"errors"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/acl"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

type aclAuthorizer interface {
	Authorize(context.Context, acl.AuthorizeRequest) error
}

type peerConfigGetter interface {
	GetPeerConfig(context.Context, adminservice.GetPeerConfigRequestObject) (adminservice.GetPeerConfigResponseObject, error)
}

type peerAuthorizer struct {
	ACL       aclAuthorizer
	Peers     peerConfigGetter
	PublicKey giznet.PublicKey
}

func (a peerAuthorizer) Authorize(ctx context.Context, request acl.AuthorizeRequest) error {
	if a.ACL == nil {
		return errors.New("acl service not configured")
	}
	err := a.ACL.Authorize(ctx, request)
	if err == nil || !errors.Is(err, acl.ErrDenied) || !a.shouldTryView(request) {
		return err
	}
	view, ok := a.peerView(ctx)
	if !ok {
		return err
	}
	request.Subject = acl.ViewSubject(view)
	viewErr := a.ACL.Authorize(ctx, request)
	if viewErr == nil {
		return nil
	}
	if errors.Is(viewErr, acl.ErrDenied) {
		return err
	}
	return viewErr
}

func (a peerAuthorizer) shouldTryView(request acl.AuthorizeRequest) bool {
	return request.Subject.Kind == acl.SubjectKindPublicKey && request.Subject.Id == a.PublicKey.String()
}

func (a peerAuthorizer) peerView(ctx context.Context) (string, bool) {
	if a.Peers == nil {
		return "", false
	}
	resp, err := a.Peers.GetPeerConfig(ctx, adminservice.GetPeerConfigRequestObject{PublicKey: a.PublicKey.String()})
	if err != nil {
		return "", false
	}
	config, ok := resp.(adminservice.GetPeerConfig200JSONResponse)
	if !ok || config.View == nil {
		return "", false
	}
	view := strings.TrimSpace(*config.View)
	return view, view != ""
}
