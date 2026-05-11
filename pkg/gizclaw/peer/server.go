package peer

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

var (
	ErrPeerNotFound      = errors.New("gear: gear not found")
	ErrPeerAlreadyExists = errors.New("gear: gear already exists")
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

type PeerManager interface {
	PeerRuntime(context.Context, giznet.PublicKey) apitypes.Runtime
	RefreshPeer(context.Context, giznet.PublicKey) (adminservice.RefreshResult, bool, error)
}

type Server struct {
	Store           kv.Store
	BuildCommit     string
	ServerPublicKey giznet.PublicKey
	PeerManager     PeerManager

	mu sync.Mutex
}

type PeerAdminService interface {
	ListPeers(context.Context, adminservice.ListPeersRequestObject) (adminservice.ListPeersResponseObject, error)
	ListPeersByCertification(context.Context, adminservice.ListPeersByCertificationRequestObject) (adminservice.ListPeersByCertificationResponseObject, error)
	ListPeersByFirmware(context.Context, adminservice.ListPeersByFirmwareRequestObject) (adminservice.ListPeersByFirmwareResponseObject, error)
	ResolvePeerByIMEI(context.Context, adminservice.ResolvePeerByIMEIRequestObject) (adminservice.ResolvePeerByIMEIResponseObject, error)
	ListPeersByLabel(context.Context, adminservice.ListPeersByLabelRequestObject) (adminservice.ListPeersByLabelResponseObject, error)
	ResolvePeerBySN(context.Context, adminservice.ResolvePeerBySNRequestObject) (adminservice.ResolvePeerBySNResponseObject, error)
	DeletePeer(context.Context, adminservice.DeletePeerRequestObject) (adminservice.DeletePeerResponseObject, error)
	GetPeer(context.Context, adminservice.GetPeerRequestObject) (adminservice.GetPeerResponseObject, error)
	GetPeerConfig(context.Context, adminservice.GetPeerConfigRequestObject) (adminservice.GetPeerConfigResponseObject, error)
	PutPeerConfig(context.Context, adminservice.PutPeerConfigRequestObject) (adminservice.PutPeerConfigResponseObject, error)
	GetPeerInfo(context.Context, adminservice.GetPeerInfoRequestObject) (adminservice.GetPeerInfoResponseObject, error)
	GetPeerRuntime(context.Context, adminservice.GetPeerRuntimeRequestObject) (adminservice.GetPeerRuntimeResponseObject, error)
	ApprovePeer(context.Context, adminservice.ApprovePeerRequestObject) (adminservice.ApprovePeerResponseObject, error)
	BlockPeer(context.Context, adminservice.BlockPeerRequestObject) (adminservice.BlockPeerResponseObject, error)
	RefreshPeer(context.Context, adminservice.RefreshPeerRequestObject) (adminservice.RefreshPeerResponseObject, error)
}

type GearService interface {
	GetConfig(context.Context, gearservice.GetConfigRequestObject) (gearservice.GetConfigResponseObject, error)
	GetInfo(context.Context, gearservice.GetInfoRequestObject) (gearservice.GetInfoResponseObject, error)
	PutInfo(context.Context, gearservice.PutInfoRequestObject) (gearservice.PutInfoResponseObject, error)
	GetRegistration(context.Context, gearservice.GetRegistrationRequestObject) (gearservice.GetRegistrationResponseObject, error)
	RegisterGear(context.Context, gearservice.RegisterGearRequestObject) (gearservice.RegisterGearResponseObject, error)
	GetRuntime(context.Context, gearservice.GetRuntimeRequestObject) (gearservice.GetRuntimeResponseObject, error)
}

type ServerPublicService interface {
	GetServerInfo(context.Context, serverpublic.GetServerInfoRequestObject) (serverpublic.GetServerInfoResponseObject, error)
}

var _ PeerAdminService = (*Server)(nil)
var _ GearService = (*Server)(nil)
var _ ServerPublicService = (*Server)(nil)

// ListPeers implements `adminservice.StrictServerInterface.ListPeers`.
func (s *Server) ListPeers(ctx context.Context, request adminservice.ListPeersRequestObject) (adminservice.ListPeersResponseObject, error) {
	cursor, limit := normalizeListParams(request.Params.Cursor, request.Params.Limit)
	items, hasNext, nextCursor, err := s.listPage(ctx, cursor, limit)
	if err != nil {
		return adminservice.ListPeers500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListPeers200JSONResponse(toAdminRegistrationList(items, hasNext, nextCursor)), nil
}

// ListPeersByCertification implements `adminservice.StrictServerInterface.ListPeersByCertification`.
func (s *Server) ListPeersByCertification(ctx context.Context, request adminservice.ListPeersByCertificationRequestObject) (adminservice.ListPeersByCertificationResponseObject, error) {
	id, err := pathUnescape(request.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	cursor, limit := normalizeListParams(request.Params.Cursor, request.Params.Limit)
	items, hasNext, nextCursor, err := s.listByCertification(ctx, apitypes.GearCertificationType(request.Type), apitypes.GearCertificationAuthority(request.Authority), id, cursor, limit)
	if err != nil {
		return adminservice.ListPeersByCertification500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListPeersByCertification200JSONResponse(toAdminRegistrationList(items, hasNext, nextCursor)), nil
}

// ListPeersByFirmware implements `adminservice.StrictServerInterface.ListPeersByFirmware`.
func (s *Server) ListPeersByFirmware(ctx context.Context, request adminservice.ListPeersByFirmwareRequestObject) (adminservice.ListPeersByFirmwareResponseObject, error) {
	depot, err := pathUnescape(request.Depot)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	cursor, limit := normalizeListParams(request.Params.Cursor, request.Params.Limit)
	items, hasNext, nextCursor, err := s.listByFirmware(ctx, depot, apitypes.GearFirmwareChannel(request.Channel), cursor, limit)
	if err != nil {
		return adminservice.ListPeersByFirmware500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListPeersByFirmware200JSONResponse(toAdminRegistrationList(items, hasNext, nextCursor)), nil
}

// ResolvePeerByIMEI implements `adminservice.StrictServerInterface.ResolvePeerByIMEI`.
func (s *Server) ResolvePeerByIMEI(ctx context.Context, request adminservice.ResolvePeerByIMEIRequestObject) (adminservice.ResolvePeerByIMEIResponseObject, error) {
	tac, err := pathUnescape(request.Tac)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	serial, err := pathUnescape(request.Serial)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	publicKey, err := s.resolveByIMEI(ctx, tac, serial)
	if err != nil {
		return adminservice.ResolvePeerByIMEI404JSONResponse(apitypes.NewErrorResponse("PEER_IMEI_NOT_FOUND", err.Error())), nil
	}
	return adminservice.ResolvePeerByIMEI200JSONResponse(adminservice.PublicKeyResponse{PublicKey: publicKey}), nil
}

// ListPeersByLabel implements `adminservice.StrictServerInterface.ListPeersByLabel`.
func (s *Server) ListPeersByLabel(ctx context.Context, request adminservice.ListPeersByLabelRequestObject) (adminservice.ListPeersByLabelResponseObject, error) {
	key, err := pathUnescape(request.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	value, err := pathUnescape(request.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	cursor, limit := normalizeListParams(request.Params.Cursor, request.Params.Limit)
	items, hasNext, nextCursor, err := s.listByLabel(ctx, key, value, cursor, limit)
	if err != nil {
		return adminservice.ListPeersByLabel500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListPeersByLabel200JSONResponse(toAdminRegistrationList(items, hasNext, nextCursor)), nil
}

// ResolvePeerBySN implements `adminservice.StrictServerInterface.ResolvePeerBySN`.
func (s *Server) ResolvePeerBySN(ctx context.Context, request adminservice.ResolvePeerBySNRequestObject) (adminservice.ResolvePeerBySNResponseObject, error) {
	sn, err := pathUnescape(request.Sn)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	publicKey, err := s.resolveBySN(ctx, sn)
	if err != nil {
		return adminservice.ResolvePeerBySN404JSONResponse(apitypes.NewErrorResponse("PEER_SN_NOT_FOUND", err.Error())), nil
	}
	return adminservice.ResolvePeerBySN200JSONResponse(adminservice.PublicKeyResponse{PublicKey: publicKey}), nil
}

// DeletePeer implements `adminservice.StrictServerInterface.DeletePeer`.
func (s *Server) DeletePeer(ctx context.Context, request adminservice.DeletePeerRequestObject) (adminservice.DeletePeerResponseObject, error) {
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.delete(ctx, publicKey)
	if err != nil {
		return adminservice.DeletePeer404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	return adminservice.DeletePeer200JSONResponse(toAdminRegistration(gear)), nil
}

// GetPeer implements `adminservice.StrictServerInterface.GetPeer`.
func (s *Server) GetPeer(ctx context.Context, request adminservice.GetPeerRequestObject) (adminservice.GetPeerResponseObject, error) {
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.get(ctx, publicKey)
	if err != nil {
		return adminservice.GetPeer404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	return adminservice.GetPeer200JSONResponse(toAdminRegistration(gear)), nil
}

// GetPeerConfig implements `adminservice.StrictServerInterface.GetPeerConfig`.
func (s *Server) GetPeerConfig(ctx context.Context, request adminservice.GetPeerConfigRequestObject) (adminservice.GetPeerConfigResponseObject, error) {
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.get(ctx, publicKey)
	if err != nil {
		return adminservice.GetPeerConfig404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	return adminservice.GetPeerConfig200JSONResponse(gear.Configuration), nil
}

// PutPeerConfig implements `adminservice.StrictServerInterface.PutPeerConfig`.
func (s *Server) PutPeerConfig(ctx context.Context, request adminservice.PutPeerConfigRequestObject) (adminservice.PutPeerConfigResponseObject, error) {
	if request.Body == nil {
		return adminservice.PutPeerConfig400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", "request body required")), nil
	}
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return adminservice.PutPeerConfig400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", err.Error())), nil
	}
	gear, err := s.putConfig(ctx, publicKey, *request.Body)
	if err != nil {
		if errors.Is(err, ErrPeerNotFound) {
			return adminservice.PutPeerConfig404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
		}
		return adminservice.PutPeerConfig400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", err.Error())), nil
	}
	return adminservice.PutPeerConfig200JSONResponse(gear.Configuration), nil
}

// GetPeerInfo implements `adminservice.StrictServerInterface.GetPeerInfo`.
func (s *Server) GetPeerInfo(ctx context.Context, request adminservice.GetPeerInfoRequestObject) (adminservice.GetPeerInfoResponseObject, error) {
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.get(ctx, publicKey)
	if err != nil {
		return adminservice.GetPeerInfo404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	return adminservice.GetPeerInfo200JSONResponse(gear.Device), nil
}

// GetPeerRuntime implements `adminservice.StrictServerInterface.GetPeerRuntime`.
func (s *Server) GetPeerRuntime(ctx context.Context, request adminservice.GetPeerRuntimeRequestObject) (adminservice.GetPeerRuntimeResponseObject, error) {
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return adminservice.GetPeerRuntime200JSONResponse(toAdminRuntime(s.peerRuntime(ctx, publicKey))), nil
}

// ApprovePeer implements `adminservice.StrictServerInterface.ApprovePeer`.
func (s *Server) ApprovePeer(ctx context.Context, request adminservice.ApprovePeerRequestObject) (adminservice.ApprovePeerResponseObject, error) {
	if request.Body == nil {
		return adminservice.ApprovePeer400JSONResponse(apitypes.NewErrorResponse("INVALID_ROLE", "request body required")), nil
	}
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return adminservice.ApprovePeer400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", err.Error())), nil
	}
	gear, err := s.approve(ctx, publicKey, apitypes.GearRole(request.Body.Role))
	if err != nil {
		return adminservice.ApprovePeer400JSONResponse(apitypes.NewErrorResponse("INVALID_ROLE", err.Error())), nil
	}
	return adminservice.ApprovePeer200JSONResponse(toAdminRegistration(gear)), nil
}

// BlockPeer implements `adminservice.StrictServerInterface.BlockPeer`.
func (s *Server) BlockPeer(ctx context.Context, request adminservice.BlockPeerRequestObject) (adminservice.BlockPeerResponseObject, error) {
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.block(ctx, publicKey)
	if err != nil {
		return adminservice.BlockPeer404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	return adminservice.BlockPeer200JSONResponse(toAdminRegistration(gear)), nil
}

// RefreshPeer implements `adminservice.StrictServerInterface.RefreshPeer`.
func (s *Server) RefreshPeer(ctx context.Context, request adminservice.RefreshPeerRequestObject) (adminservice.RefreshPeerResponseObject, error) {
	if s.PeerManager == nil {
		return adminservice.RefreshPeer502JSONResponse(apitypes.NewErrorResponse("DEVICE_REFRESH_FAILED", "refresh provider not configured")), nil
	}
	publicKey, err := parsePublicKeyParam(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	result, online, err := s.PeerManager.RefreshPeer(ctx, publicKey)
	if err != nil {
		switch {
		case errors.Is(err, ErrPeerNotFound):
			return adminservice.RefreshPeer404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
		case !online:
			return adminservice.RefreshPeer409JSONResponse(apitypes.NewErrorResponse("DEVICE_OFFLINE", err.Error())), nil
		default:
			return adminservice.RefreshPeer502JSONResponse(apitypes.NewErrorResponse("DEVICE_REFRESH_FAILED", err.Error())), nil
		}
	}
	return adminservice.RefreshPeer200JSONResponse(result), nil
}

// GetConfig implements `gearservice.StrictServerInterface.GetConfig`.
func (s *Server) GetConfig(ctx context.Context, _ gearservice.GetConfigRequestObject) (gearservice.GetConfigResponseObject, error) {
	gear, err := s.get(ctx, gearservice.CallerPublicKey(ctx))
	if err != nil {
		return gearservice.GetConfig404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	cfg, err := toGearConfiguration(gear.Configuration)
	if err != nil {
		return getConfig500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return gearservice.GetConfig200JSONResponse(cfg), nil
}

// GetInfo implements `gearservice.StrictServerInterface.GetInfo`.
func (s *Server) GetInfo(ctx context.Context, _ gearservice.GetInfoRequestObject) (gearservice.GetInfoResponseObject, error) {
	gear, err := s.get(ctx, gearservice.CallerPublicKey(ctx))
	if err != nil {
		return gearservice.GetInfo404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	info, err := toGearDeviceInfo(gear.Device)
	if err != nil {
		return getInfo500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return gearservice.GetInfo200JSONResponse(info), nil
}

// PutInfo implements `gearservice.StrictServerInterface.PutInfo`.
func (s *Server) PutInfo(ctx context.Context, request gearservice.PutInfoRequestObject) (gearservice.PutInfoResponseObject, error) {
	if request.Body == nil {
		return gearservice.PutInfo400JSONResponse(apitypes.NewErrorResponse("INVALID_DEVICE_INFO", "request body required")), nil
	}
	info, err := toAdminDeviceInfo(*request.Body)
	if err != nil {
		return gearservice.PutInfo400JSONResponse(apitypes.NewErrorResponse("INVALID_DEVICE_INFO", err.Error())), nil
	}
	gear, err := s.putInfo(ctx, gearservice.CallerPublicKey(ctx), info)
	if err != nil {
		return gearservice.PutInfo404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	out, err := toGearDeviceInfo(gear.Device)
	if err != nil {
		return putInfo500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return gearservice.PutInfo200JSONResponse(out), nil
}

// RegisterGear implements `gearservice.StrictServerInterface.RegisterGear`.
func (s *Server) RegisterGear(ctx context.Context, request gearservice.RegisterGearRequestObject) (gearservice.RegisterGearResponseObject, error) {
	if request.Body == nil {
		return gearservice.RegisterGear400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", "request body required")), nil
	}
	gear, err := s.register(ctx, gearservice.CallerPublicKey(ctx), *request.Body)
	if err != nil {
		if errors.Is(err, ErrPeerAlreadyExists) {
			return gearservice.RegisterGear409JSONResponse(apitypes.NewErrorResponse("PEER_ALREADY_EXISTS", err.Error())), nil
		}
		return gearservice.RegisterGear400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", err.Error())), nil
	}
	out, err := toPublicRegistrationResult(gear)
	if err != nil {
		return registerGear500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", err.Error())), nil
	}
	return gearservice.RegisterGear200JSONResponse(out), nil
}

// GetRegistration implements `gearservice.StrictServerInterface.GetRegistration`.
func (s *Server) GetRegistration(ctx context.Context, _ gearservice.GetRegistrationRequestObject) (gearservice.GetRegistrationResponseObject, error) {
	gear, err := s.get(ctx, gearservice.CallerPublicKey(ctx))
	if err != nil {
		return gearservice.GetRegistration404JSONResponse(apitypes.NewErrorResponse("PEER_NOT_FOUND", err.Error())), nil
	}
	return gearservice.GetRegistration200JSONResponse(toGearRegistration(gear)), nil
}

// GetRuntime implements `gearservice.StrictServerInterface.GetRuntime`.
func (s *Server) GetRuntime(ctx context.Context, _ gearservice.GetRuntimeRequestObject) (gearservice.GetRuntimeResponseObject, error) {
	return gearservice.GetRuntime200JSONResponse(s.peerRuntime(ctx, gearservice.CallerPublicKey(ctx))), nil
}

// GetServerInfo implements `serverpublic.StrictServerInterface.GetServerInfo`.
func (s *Server) GetServerInfo(_ context.Context, _ serverpublic.GetServerInfoRequestObject) (serverpublic.GetServerInfoResponseObject, error) {
	return serverpublic.GetServerInfo200JSONResponse(apitypes.ServerInfo{
		BuildCommit: s.BuildCommit,
		PublicKey:   s.ServerPublicKey.String(),
		ServerTime:  time.Now().UnixMilli(),
	}), nil
}

func pathUnescape(value string) (string, error) {
	return url.PathUnescape(value)
}

func parsePublicKeyParam(value string) (giznet.PublicKey, error) {
	text, err := pathUnescape(value)
	if err != nil {
		return giznet.PublicKey{}, err
	}
	return publicKeyFromText(text)
}

func normalizeListParams(cursor *adminservice.Cursor, limit *adminservice.Limit) (string, int) {
	nextCursor := ""
	if cursor != nil {
		nextCursor = string(*cursor)
	}
	nextLimit := defaultListLimit
	if limit != nil {
		nextLimit = int(*limit)
	}
	if nextLimit <= 0 {
		nextLimit = defaultListLimit
	}
	if nextLimit > maxListLimit {
		nextLimit = maxListLimit
	}
	return nextCursor, nextLimit
}
