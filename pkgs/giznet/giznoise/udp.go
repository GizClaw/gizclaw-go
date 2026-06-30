package giznoise

import (
	"net"
	"time"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/internal/core"
)

type HostInfo struct {
	PublicKey giznet.PublicKey
	Addr      *net.UDPAddr
	PeerCount int
	RxBytes   uint64
	TxBytes   uint64
	LastSeen  time.Time

	DroppedOutputPackets  uint64
	DroppedDecryptPackets uint64
	DroppedInboundPackets uint64
	RPCRouteErrors        uint64
	KCPOutputErrors       uint64
	DroppedPeerEvents     uint64
}

type UDP struct {
	inner *core.UDP
}

func (u *UDP) HostInfo() *HostInfo {
	if u == nil || u.inner == nil {
		return nil
	}
	return fromCoreHostInfo(u.inner.HostInfo())
}

func (u *UDP) SetPeerEndpoint(pk giznet.PublicKey, endpoint *net.UDPAddr) {
	u.inner.SetPeerEndpoint(toNoisePublicKey(pk), endpoint)
}

func (u *UDP) Connect(pk giznet.PublicKey) error {
	return u.inner.Connect(toNoisePublicKey(pk))
}

func (u *UDP) PeerInfo(pk giznet.PublicKey) *giznet.PeerInfo {
	return fromCorePeerInfo(u.inner.PeerInfo(toNoisePublicKey(pk)))
}

func (u *UDP) PeerServiceMux(pk giznet.PublicKey) (*core.ServiceMux, error) {
	return u.inner.PeerServiceMux(toNoisePublicKey(pk))
}

func (u *UDP) WriteTo(pk giznet.PublicKey, data []byte) error {
	return u.inner.WriteTo(toNoisePublicKey(pk), data)
}

func (u *UDP) ReadFrom(buf []byte) (giznet.PublicKey, int, error) {
	pk, n, err := u.inner.ReadFrom(buf)
	return fromNoisePublicKey(pk), n, err
}

func (u *UDP) ReadPacket(buf []byte) (giznet.PublicKey, byte, int, error) {
	pk, proto, n, err := u.inner.ReadPacket(buf)
	return fromNoisePublicKey(pk), proto, n, err
}

func (u *UDP) Close() error {
	return u.inner.Close()
}

func fromCoreHostInfo(info *core.HostInfo) *HostInfo {
	if info == nil {
		return nil
	}
	return &HostInfo{
		PublicKey:             fromNoisePublicKey(info.PublicKey),
		Addr:                  info.Addr,
		PeerCount:             info.PeerCount,
		RxBytes:               info.RxBytes,
		TxBytes:               info.TxBytes,
		LastSeen:              info.LastSeen,
		DroppedOutputPackets:  info.DroppedOutputPackets,
		DroppedDecryptPackets: info.DroppedDecryptPackets,
		DroppedInboundPackets: info.DroppedInboundPackets,
		RPCRouteErrors:        info.RPCRouteErrors,
		KCPOutputErrors:       info.KCPOutputErrors,
		DroppedPeerEvents:     info.DroppedPeerEvents,
	}
}
