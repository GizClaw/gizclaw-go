package gizclaw

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/adminservice"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/gearservice"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/peerpublic"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/rpc"
	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/serverpublic"
	"github.com/giztoy/giztoy-go/pkg/giznet"
	"github.com/giztoy/giztoy-go/pkg/giznet/gizhttp"
	"github.com/gofiber/fiber/v2"
)

var _ peerpublic.StrictServerInterface = (*Client)(nil)

// Client holds device-side peer client configuration.
type Client struct {
	KeyPair *giznet.KeyPair

	Device gearservice.DeviceInfo

	mu       sync.RWMutex
	listener *giznet.Listener
	conn     *giznet.Conn
	serverPK giznet.PublicKey
}

// DialAndServe establishes the peer connection and serves peer public handlers.
func (c *Client) DialAndServe(serverPK giznet.PublicKey, serverAddr string, opts ...giznet.Option) error {
	if c == nil {
		return fmt.Errorf("gizclaw: nil client")
	}
	if c.KeyPair == nil {
		return fmt.Errorf("gizclaw: nil key pair")
	}
	if serverAddr == "" {
		return fmt.Errorf("gizclaw: empty server addr")
	}
	c.mu.RLock()
	alreadyStarted := c.listener != nil || c.conn != nil
	c.mu.RUnlock()
	if alreadyStarted {
		return fmt.Errorf("gizclaw: client already started")
	}

	all := append([]giznet.Option{
		giznet.WithBindAddr("127.0.0.1:0"),
		giznet.WithAllowUnknown(true),
		giznet.WithServiceMuxConfig(giznet.ServiceMuxConfig{
			OnNewService: func(_ giznet.PublicKey, service uint64) bool {
				return service == ServiceServerPublic || service == ServicePeerPublic
			},
		}),
	}, opts...)
	l, err := giznet.Listen(c.KeyPair, all...)
	if err != nil {
		return fmt.Errorf("gizclaw: listen: %w", err)
	}
	c.mu.Lock()
	c.listener = l
	c.serverPK = serverPK
	c.mu.Unlock()

	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		c.mu.Lock()
		if c.listener == l {
			c.listener = nil
			c.serverPK = giznet.PublicKey{}
		}
		c.mu.Unlock()
		_ = l.Close()
		return fmt.Errorf("gizclaw: resolve addr: %w", err)
	}

	conn, err := l.Dial(serverPK, udpAddr)
	if err != nil {
		c.mu.Lock()
		if c.listener == l {
			c.listener = nil
			c.serverPK = giznet.PublicKey{}
		}
		c.mu.Unlock()
		_ = l.Close()
		return fmt.Errorf("gizclaw: dial: %w", err)
	}
	c.mu.Lock()
	if c.listener != l {
		c.mu.Unlock()
		_ = conn.Close()
		_ = l.Close()
		return fmt.Errorf("gizclaw: client closed during dial")
	}
	c.conn = conn
	c.mu.Unlock()

	if err := c.servePeerPublic(); err != nil {
		_ = c.Close()
		return err
	}
	return nil
}

// Close releases all resources including the underlying UDP socket.
func (c *Client) Close() error {
	c.mu.Lock()
	conn := c.conn
	listener := c.listener
	c.conn = nil
	c.listener = nil
	c.serverPK = giznet.PublicKey{}
	c.mu.Unlock()

	var err error
	if conn != nil {
		if closeErr := conn.Close(); closeErr != nil {
			err = closeErr
		}
	}
	if listener != nil {
		if closeErr := listener.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (c *Client) GetInfo(_ context.Context, _ peerpublic.GetInfoRequestObject) (peerpublic.GetInfoResponseObject, error) {
	return peerpublic.GetInfo200JSONResponse(gearDeviceToPeerRefreshInfo(c.Device)), nil
}

func (c *Client) GetIdentifiers(_ context.Context, _ peerpublic.GetIdentifiersRequestObject) (peerpublic.GetIdentifiersResponseObject, error) {
	return peerpublic.GetIdentifiers200JSONResponse(gearDeviceToPeerRefreshIdentifiers(c.Device)), nil
}

func (c *Client) GetVersion(_ context.Context, _ peerpublic.GetVersionRequestObject) (peerpublic.GetVersionResponseObject, error) {
	return peerpublic.GetVersion200JSONResponse(gearDeviceToPeerRefreshVersion(c.Device)), nil
}

// HTTPClient returns an HTTP client bound to a peer service.
func (c *Client) HTTPClient(service uint64) *http.Client {
	return gizhttp.NewClient(c.PeerConn(), service)
}

func (c *Client) ServerAdminClient() (*adminservice.ClientWithResponses, error) {
	return adminservice.NewClientWithResponses(
		"http://gizclaw",
		adminservice.WithHTTPClient(c.HTTPClient(ServiceAdmin)),
	)
}

func (c *Client) GearServiceClient() (*gearservice.ClientWithResponses, error) {
	return gearservice.NewClientWithResponses(
		"http://gizclaw",
		gearservice.WithHTTPClient(c.HTTPClient(ServiceGear)),
	)
}

func (c *Client) ServerPublicClient() (*serverpublic.ClientWithResponses, error) {
	return serverpublic.NewClientWithResponses(
		"http://gizclaw",
		serverpublic.WithHTTPClient(c.HTTPClient(ServiceServerPublic)),
	)
}

func (c *Client) PeerPublicClient() (*peerpublic.ClientWithResponses, error) {
	return peerpublic.NewClientWithResponses(
		"http://gizclaw",
		peerpublic.WithHTTPClient(c.HTTPClient(ServicePeerPublic)),
	)
}

// RPCClient opens a JSON-RPC session over the peer RPC service.
func (c *Client) RPCClient() (*rpc.Client, error) {
	conn := c.PeerConn()
	if conn == nil {
		return nil, fmt.Errorf("gizclaw: client is not connected")
	}
	stream, err := conn.Dial(ServiceRPC)
	if err != nil {
		return nil, fmt.Errorf("gizclaw: dial rpc stream: %w", err)
	}
	return rpc.NewClient(stream), nil
}

// PeerConn returns the underlying peer connection.
func (c *Client) PeerConn() *giznet.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// ServerPublicKey returns the expected remote server public key.
func (c *Client) ServerPublicKey() giznet.PublicKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverPK
}

// servePeerPublic runs the device-side peer public HTTP service on ServicePeerPublic.
func (c *Client) servePeerPublic() error {
	if c == nil {
		return fmt.Errorf("gizclaw: nil client")
	}
	if c.conn == nil {
		return fmt.Errorf("gizclaw: client is not connected")
	}
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	peerpublic.RegisterHandlers(app, peerpublic.NewStrictHandler(c, nil))

	server := gizhttp.NewServer(c.conn, ServicePeerPublic, fiberHTTPHandler(app))
	return server.Serve()
}

func gearDeviceToPeerRefreshInfo(in gearservice.DeviceInfo) peerpublic.RefreshInfo {
	out := peerpublic.RefreshInfo{}
	if in.Name != nil {
		out.Name = in.Name
	}
	if in.Hardware != nil {
		out.Manufacturer = in.Hardware.Manufacturer
		out.Model = in.Hardware.Model
		out.HardwareRevision = in.Hardware.HardwareRevision
	}
	return out
}

func gearToPeerGearIMEI(in gearservice.GearIMEI) peerpublic.GearIMEI {
	out := peerpublic.GearIMEI{
		Tac:    in.Tac,
		Serial: in.Serial,
	}
	out.Name = in.Name
	return out
}

func gearToPeerGearLabel(in gearservice.GearLabel) peerpublic.GearLabel {
	return peerpublic.GearLabel{
		Key:   in.Key,
		Value: in.Value,
	}
}

func gearDeviceToPeerRefreshIdentifiers(in gearservice.DeviceInfo) peerpublic.RefreshIdentifiers {
	out := peerpublic.RefreshIdentifiers{}
	out.Sn = in.Sn
	if in.Hardware != nil {
		if in.Hardware.Imeis != nil {
			items := make([]peerpublic.GearIMEI, len(*in.Hardware.Imeis))
			for i := range *in.Hardware.Imeis {
				items[i] = gearToPeerGearIMEI((*in.Hardware.Imeis)[i])
			}
			out.Imeis = &items
		}
		if in.Hardware.Labels != nil {
			items := make([]peerpublic.GearLabel, len(*in.Hardware.Labels))
			for i := range *in.Hardware.Labels {
				items[i] = gearToPeerGearLabel((*in.Hardware.Labels)[i])
			}
			out.Labels = &items
		}
	}
	return out
}

func gearDeviceToPeerRefreshVersion(in gearservice.DeviceInfo) peerpublic.RefreshVersion {
	out := peerpublic.RefreshVersion{}
	if in.Hardware != nil {
		out.Depot = in.Hardware.Depot
		out.FirmwareSemver = in.Hardware.FirmwareSemver
	}
	return out
}
