package gear

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

var (
	ErrGearNotFound      = errors.New("gear: gear not found")
	ErrGearAlreadyExists = errors.New("gear: gear already exists")
)

type PeerManager interface {
	PeerRuntime(context.Context, string) gearservice.Runtime
	RefreshGear(context.Context, string) (gearservice.RefreshResult, bool, error)
}

type Server struct {
	Store              kv.Store
	RegistrationTokens map[string]gearservice.GearRole
	BuildCommit        string
	ServerPublicKey    string
	PeerManager        PeerManager

	mu sync.Mutex
}

type GearsAdminService interface {
	ListGears(context.Context, adminservice.ListGearsRequestObject) (adminservice.ListGearsResponseObject, error)
	ListByCertification(context.Context, adminservice.ListByCertificationRequestObject) (adminservice.ListByCertificationResponseObject, error)
	ListByFirmware(context.Context, adminservice.ListByFirmwareRequestObject) (adminservice.ListByFirmwareResponseObject, error)
	ResolveByIMEI(context.Context, adminservice.ResolveByIMEIRequestObject) (adminservice.ResolveByIMEIResponseObject, error)
	ListByLabel(context.Context, adminservice.ListByLabelRequestObject) (adminservice.ListByLabelResponseObject, error)
	ResolveBySN(context.Context, adminservice.ResolveBySNRequestObject) (adminservice.ResolveBySNResponseObject, error)
	DeleteGear(context.Context, adminservice.DeleteGearRequestObject) (adminservice.DeleteGearResponseObject, error)
	GetGear(context.Context, adminservice.GetGearRequestObject) (adminservice.GetGearResponseObject, error)
	GetGearConfig(context.Context, adminservice.GetGearConfigRequestObject) (adminservice.GetGearConfigResponseObject, error)
	PutGearConfig(context.Context, adminservice.PutGearConfigRequestObject) (adminservice.PutGearConfigResponseObject, error)
	GetGearInfo(context.Context, adminservice.GetGearInfoRequestObject) (adminservice.GetGearInfoResponseObject, error)
	GetGearRuntime(context.Context, adminservice.GetGearRuntimeRequestObject) (adminservice.GetGearRuntimeResponseObject, error)
	ApproveGear(context.Context, adminservice.ApproveGearRequestObject) (adminservice.ApproveGearResponseObject, error)
	BlockGear(context.Context, adminservice.BlockGearRequestObject) (adminservice.BlockGearResponseObject, error)
	RefreshGear(context.Context, adminservice.RefreshGearRequestObject) (adminservice.RefreshGearResponseObject, error)
}

type GearsGearService interface {
	GetConfig(context.Context, gearservice.GetConfigRequestObject) (gearservice.GetConfigResponseObject, error)
	GetInfo(context.Context, gearservice.GetInfoRequestObject) (gearservice.GetInfoResponseObject, error)
	PutInfo(context.Context, gearservice.PutInfoRequestObject) (gearservice.PutInfoResponseObject, error)
	GetRegistration(context.Context, gearservice.GetRegistrationRequestObject) (gearservice.GetRegistrationResponseObject, error)
	GetRuntime(context.Context, gearservice.GetRuntimeRequestObject) (gearservice.GetRuntimeResponseObject, error)
}

type GearsServerPublic interface {
	RegisterGear(context.Context, serverpublic.RegisterGearRequestObject) (serverpublic.RegisterGearResponseObject, error)
	GetServerInfo(context.Context, serverpublic.GetServerInfoRequestObject) (serverpublic.GetServerInfoResponseObject, error)
}

var _ GearsAdminService = (*Server)(nil)
var _ GearsGearService = (*Server)(nil)
var _ GearsServerPublic = (*Server)(nil)

// ListGears implements `adminservice.StrictServerInterface.ListGears`.
func (s *Server) ListGears(ctx context.Context, _ adminservice.ListGearsRequestObject) (adminservice.ListGearsResponseObject, error) {
	items, err := s.list(ctx)
	if err != nil {
		return adminservice.ListGears500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListGears200JSONResponse(toAdminRegistrationList(items)), nil
}

// ListByCertification implements `adminservice.StrictServerInterface.ListByCertification`.
func (s *Server) ListByCertification(ctx context.Context, request adminservice.ListByCertificationRequestObject) (adminservice.ListByCertificationResponseObject, error) {
	id, err := pathUnescape(request.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	items, err := s.listByCertification(ctx, gearservice.GearCertificationType(request.Type), gearservice.GearCertificationAuthority(request.Authority), id)
	if err != nil {
		return adminservice.ListByCertification500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListByCertification200JSONResponse(toAdminRegistrationList(items)), nil
}

// ListByFirmware implements `adminservice.StrictServerInterface.ListByFirmware`.
func (s *Server) ListByFirmware(ctx context.Context, request adminservice.ListByFirmwareRequestObject) (adminservice.ListByFirmwareResponseObject, error) {
	depot, err := pathUnescape(request.Depot)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	items, err := s.listByFirmware(ctx, depot, gearservice.GearFirmwareChannel(request.Channel))
	if err != nil {
		return adminservice.ListByFirmware500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListByFirmware200JSONResponse(toAdminRegistrationList(items)), nil
}

// ResolveByIMEI implements `adminservice.StrictServerInterface.ResolveByIMEI`.
func (s *Server) ResolveByIMEI(ctx context.Context, request adminservice.ResolveByIMEIRequestObject) (adminservice.ResolveByIMEIResponseObject, error) {
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
		return adminservice.ResolveByIMEI404JSONResponse(adminError("GEAR_IMEI_NOT_FOUND", err.Error())), nil
	}
	return adminservice.ResolveByIMEI200JSONResponse(adminservice.PublicKeyResponse{PublicKey: publicKey}), nil
}

// ListByLabel implements `adminservice.StrictServerInterface.ListByLabel`.
func (s *Server) ListByLabel(ctx context.Context, request adminservice.ListByLabelRequestObject) (adminservice.ListByLabelResponseObject, error) {
	key, err := pathUnescape(request.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	value, err := pathUnescape(request.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	items, err := s.listByLabel(ctx, key, value)
	if err != nil {
		return adminservice.ListByLabel500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.ListByLabel200JSONResponse(toAdminRegistrationList(items)), nil
}

// ResolveBySN implements `adminservice.StrictServerInterface.ResolveBySN`.
func (s *Server) ResolveBySN(ctx context.Context, request adminservice.ResolveBySNRequestObject) (adminservice.ResolveBySNResponseObject, error) {
	sn, err := pathUnescape(request.Sn)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	publicKey, err := s.resolveBySN(ctx, sn)
	if err != nil {
		return adminservice.ResolveBySN404JSONResponse(adminError("GEAR_SN_NOT_FOUND", err.Error())), nil
	}
	return adminservice.ResolveBySN200JSONResponse(adminservice.PublicKeyResponse{PublicKey: publicKey}), nil
}

// DeleteGear implements `adminservice.StrictServerInterface.DeleteGear`.
func (s *Server) DeleteGear(ctx context.Context, request adminservice.DeleteGearRequestObject) (adminservice.DeleteGearResponseObject, error) {
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.delete(ctx, publicKey)
	if err != nil {
		return adminservice.DeleteGear404JSONResponse(adminError("GEAR_NOT_FOUND", err.Error())), nil
	}
	return adminservice.DeleteGear200JSONResponse(toAdminRegistration(gear)), nil
}

// GetGear implements `adminservice.StrictServerInterface.GetGear`.
func (s *Server) GetGear(ctx context.Context, request adminservice.GetGearRequestObject) (adminservice.GetGearResponseObject, error) {
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.get(ctx, publicKey)
	if err != nil {
		return adminservice.GetGear404JSONResponse(adminError("GEAR_NOT_FOUND", err.Error())), nil
	}
	return adminservice.GetGear200JSONResponse(toAdminRegistration(gear)), nil
}

// GetGearConfig implements `adminservice.StrictServerInterface.GetGearConfig`.
func (s *Server) GetGearConfig(ctx context.Context, request adminservice.GetGearConfigRequestObject) (adminservice.GetGearConfigResponseObject, error) {
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.get(ctx, publicKey)
	if err != nil {
		return adminservice.GetGearConfig404JSONResponse(adminError("GEAR_NOT_FOUND", err.Error())), nil
	}
	cfg, err := toAdminConfiguration(gear.Configuration)
	if err != nil {
		return getGearConfig500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.GetGearConfig200JSONResponse(cfg), nil
}

// PutGearConfig implements `adminservice.StrictServerInterface.PutGearConfig`.
func (s *Server) PutGearConfig(ctx context.Context, request adminservice.PutGearConfigRequestObject) (adminservice.PutGearConfigResponseObject, error) {
	if request.Body == nil {
		return adminservice.PutGearConfig400JSONResponse(adminError("INVALID_PARAMS", "request body required")), nil
	}
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return adminservice.PutGearConfig400JSONResponse(adminError("INVALID_PARAMS", err.Error())), nil
	}
	body, err := convertViaJSON[gearservice.Configuration](*request.Body)
	if err != nil {
		return adminservice.PutGearConfig400JSONResponse(adminError("INVALID_PARAMS", err.Error())), nil
	}
	gear, err := s.putConfig(ctx, publicKey, body)
	if err != nil {
		if errors.Is(err, ErrGearNotFound) {
			return adminservice.PutGearConfig404JSONResponse(adminError("GEAR_NOT_FOUND", err.Error())), nil
		}
		return adminservice.PutGearConfig400JSONResponse(adminError("INVALID_PARAMS", err.Error())), nil
	}
	cfg, err := toAdminConfiguration(gear.Configuration)
	if err != nil {
		return putGearConfig500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.PutGearConfig200JSONResponse(cfg), nil
}

// GetGearInfo implements `adminservice.StrictServerInterface.GetGearInfo`.
func (s *Server) GetGearInfo(ctx context.Context, request adminservice.GetGearInfoRequestObject) (adminservice.GetGearInfoResponseObject, error) {
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.get(ctx, publicKey)
	if err != nil {
		return adminservice.GetGearInfo404JSONResponse(adminError("GEAR_NOT_FOUND", err.Error())), nil
	}
	info, err := toAdminDeviceInfo(gear.Device)
	if err != nil {
		return getGearInfo500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.GetGearInfo200JSONResponse(info), nil
}

// GetGearRuntime implements `adminservice.StrictServerInterface.GetGearRuntime`.
func (s *Server) GetGearRuntime(ctx context.Context, request adminservice.GetGearRuntimeRequestObject) (adminservice.GetGearRuntimeResponseObject, error) {
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return adminservice.GetGearRuntime200JSONResponse(toAdminRuntime(s.peerRuntime(ctx, publicKey))), nil
}

// ApproveGear implements `adminservice.StrictServerInterface.ApproveGear`.
func (s *Server) ApproveGear(ctx context.Context, request adminservice.ApproveGearRequestObject) (adminservice.ApproveGearResponseObject, error) {
	if request.Body == nil {
		return adminservice.ApproveGear400JSONResponse(adminError("INVALID_ROLE", "request body required")), nil
	}
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return adminservice.ApproveGear400JSONResponse(adminError("INVALID_PARAMS", err.Error())), nil
	}
	gear, err := s.approve(ctx, publicKey, gearservice.GearRole(request.Body.Role))
	if err != nil {
		return adminservice.ApproveGear400JSONResponse(adminError("INVALID_ROLE", err.Error())), nil
	}
	return adminservice.ApproveGear200JSONResponse(toAdminRegistration(gear)), nil
}

// BlockGear implements `adminservice.StrictServerInterface.BlockGear`.
func (s *Server) BlockGear(ctx context.Context, request adminservice.BlockGearRequestObject) (adminservice.BlockGearResponseObject, error) {
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	gear, err := s.block(ctx, publicKey)
	if err != nil {
		return adminservice.BlockGear404JSONResponse(adminError("GEAR_NOT_FOUND", err.Error())), nil
	}
	return adminservice.BlockGear200JSONResponse(toAdminRegistration(gear)), nil
}

// RefreshGear implements `adminservice.StrictServerInterface.RefreshGear`.
func (s *Server) RefreshGear(ctx context.Context, request adminservice.RefreshGearRequestObject) (adminservice.RefreshGearResponseObject, error) {
	if s.PeerManager == nil {
		return adminservice.RefreshGear502JSONResponse(adminError("DEVICE_REFRESH_FAILED", "refresh provider not configured")), nil
	}
	publicKey, err := pathUnescape(string(request.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	result, online, err := s.PeerManager.RefreshGear(ctx, publicKey)
	if err != nil {
		switch {
		case errors.Is(err, ErrGearNotFound):
			return adminservice.RefreshGear404JSONResponse(adminError("GEAR_NOT_FOUND", err.Error())), nil
		case !online:
			return adminservice.RefreshGear409JSONResponse(adminError("DEVICE_OFFLINE", err.Error())), nil
		default:
			return adminservice.RefreshGear502JSONResponse(adminError("DEVICE_REFRESH_FAILED", err.Error())), nil
		}
	}
	out, err := toAdminRefreshResult(result)
	if err != nil {
		return refreshGear500JSONResponse(adminError("INTERNAL_ERROR", err.Error())), nil
	}
	return adminservice.RefreshGear200JSONResponse(out), nil
}

// GetConfig implements `gearservice.StrictServerInterface.GetConfig`.
func (s *Server) GetConfig(ctx context.Context, _ gearservice.GetConfigRequestObject) (gearservice.GetConfigResponseObject, error) {
	gear, err := s.get(ctx, gearservice.CallerPublicKey(ctx))
	if err != nil {
		return gearservice.GetConfig404JSONResponse(gearError("GEAR_NOT_FOUND", err.Error())), nil
	}
	return gearservice.GetConfig200JSONResponse(gear.Configuration), nil
}

// GetInfo implements `gearservice.StrictServerInterface.GetInfo`.
func (s *Server) GetInfo(ctx context.Context, _ gearservice.GetInfoRequestObject) (gearservice.GetInfoResponseObject, error) {
	gear, err := s.get(ctx, gearservice.CallerPublicKey(ctx))
	if err != nil {
		return gearservice.GetInfo404JSONResponse(gearError("GEAR_NOT_FOUND", err.Error())), nil
	}
	return gearservice.GetInfo200JSONResponse(gear.Device), nil
}

// PutInfo implements `gearservice.StrictServerInterface.PutInfo`.
func (s *Server) PutInfo(ctx context.Context, request gearservice.PutInfoRequestObject) (gearservice.PutInfoResponseObject, error) {
	if request.Body == nil {
		return gearservice.PutInfo400JSONResponse(gearError("INVALID_DEVICE_INFO", "request body required")), nil
	}
	gear, err := s.putInfo(ctx, gearservice.CallerPublicKey(ctx), *request.Body)
	if err != nil {
		return gearservice.PutInfo404JSONResponse(gearError("GEAR_NOT_FOUND", err.Error())), nil
	}
	return gearservice.PutInfo200JSONResponse(gear.Device), nil
}

// RegisterGear implements `serverpublic.StrictServerInterface.RegisterGear`.
func (s *Server) RegisterGear(ctx context.Context, request serverpublic.RegisterGearRequestObject) (serverpublic.RegisterGearResponseObject, error) {
	if request.Body == nil {
		return serverpublic.RegisterGear400JSONResponse(publicError("INVALID_PARAMS", "request body required")), nil
	}
	body, err := toGearRegistrationRequest(*request.Body)
	if err != nil {
		return serverpublic.RegisterGear400JSONResponse(publicError("INVALID_PARAMS", err.Error())), nil
	}
	body.PublicKey = serverpublic.CallerPublicKey(ctx)
	result, err := s.register(ctx, body)
	if err != nil {
		if errors.Is(err, ErrGearAlreadyExists) {
			return serverpublic.RegisterGear409JSONResponse(publicError("GEAR_ALREADY_EXISTS", err.Error())), nil
		}
		return serverpublic.RegisterGear400JSONResponse(publicError("INVALID_PARAMS", err.Error())), nil
	}
	out, err := toPublicRegistrationResult(result)
	if err != nil {
		return registerGear500JSONResponse(publicError("INTERNAL_ERROR", err.Error())), nil
	}
	return serverpublic.RegisterGear200JSONResponse(out), nil
}

// GetRegistration implements `gearservice.StrictServerInterface.GetRegistration`.
func (s *Server) GetRegistration(ctx context.Context, _ gearservice.GetRegistrationRequestObject) (gearservice.GetRegistrationResponseObject, error) {
	gear, err := s.get(ctx, gearservice.CallerPublicKey(ctx))
	if err != nil {
		return gearservice.GetRegistration404JSONResponse(gearError("GEAR_NOT_FOUND", err.Error())), nil
	}
	return gearservice.GetRegistration200JSONResponse(toGearRegistration(gear)), nil
}

// GetRuntime implements `gearservice.StrictServerInterface.GetRuntime`.
func (s *Server) GetRuntime(ctx context.Context, _ gearservice.GetRuntimeRequestObject) (gearservice.GetRuntimeResponseObject, error) {
	return gearservice.GetRuntime200JSONResponse(s.peerRuntime(ctx, gearservice.CallerPublicKey(ctx))), nil
}

// GetServerInfo implements `serverpublic.StrictServerInterface.GetServerInfo`.
func (s *Server) GetServerInfo(_ context.Context, _ serverpublic.GetServerInfoRequestObject) (serverpublic.GetServerInfoResponseObject, error) {
	return serverpublic.GetServerInfo200JSONResponse(serverpublic.ServerInfo{
		BuildCommit: s.BuildCommit,
		PublicKey:   s.ServerPublicKey,
		ServerTime:  time.Now().UnixMilli(),
	}), nil
}

func pathUnescape(value string) (string, error) {
	return url.PathUnescape(value)
}
