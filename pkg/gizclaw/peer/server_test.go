package peer

import (
	"context"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

type stubPeerManager struct {
	runtime       apitypes.Runtime
	refreshResult adminservice.RefreshResult
	refreshOnline bool
	refreshErr    error
}

func (m stubPeerManager) PeerRuntime(context.Context, giznet.PublicKey) apitypes.Runtime {
	return m.runtime
}

func (m stubPeerManager) RefreshPeer(context.Context, giznet.PublicKey) (adminservice.RefreshResult, bool, error) {
	return m.refreshResult, m.refreshOnline, m.refreshErr
}

func TestServerGearserviceHandlers(t *testing.T) {
	server := &Server{
		Store: mustBadgerInMemory(t, nil),
	}

	peerKey := giznet.PublicKey{1}
	peerPublicKey := peerKey.String()
	ctx := gearservice.WithCallerPublicKey(context.Background(), peerKey)
	sn := "sn-gear"
	depot := "depot-gear"
	tac := "12345678"
	serial := "87654321"
	labelKey := "region"
	labelValue := "cn"
	certID := "cert-gear"

	_, err := server.RegisterGear(ctx, gearservice.RegisterGearRequestObject{
		Body: &gearservice.RegisterGearJSONRequestBody{
			Device: apitypes.DeviceInfo{
				Sn: &sn,
				Hardware: &apitypes.HardwareInfo{
					Depot:  &depot,
					Imeis:  &[]apitypes.GearIMEI{{Tac: tac, Serial: serial}},
					Labels: &[]apitypes.GearLabel{{Key: labelKey, Value: labelValue}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterGear error: %v", err)
	}

	getResp, err := server.GetPeer(ctx, adminservice.GetPeerRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
	})
	if err != nil {
		t.Fatalf("GetPeer error: %v", err)
	}
	getRegistered, ok := getResp.(adminservice.GetPeer200JSONResponse)
	if !ok {
		t.Fatalf("GetPeer response type = %T", getResp)
	}
	if getRegistered.PublicKey != peerPublicKey {
		t.Fatalf("GetPeer = %+v", getRegistered)
	}

	listResp, err := server.ListPeers(ctx, adminservice.ListPeersRequestObject{})
	if err != nil {
		t.Fatalf("ListPeers error: %v", err)
	}
	listed, ok := listResp.(adminservice.ListPeers200JSONResponse)
	if !ok {
		t.Fatalf("ListPeers response type = %T", listResp)
	}
	if len(listed.Items) != 1 || listed.Items[0].PublicKey != peerPublicKey {
		t.Fatalf("ListPeers items = %+v", listed.Items)
	}

	stable := apitypes.GearFirmwareChannel("stable")
	adminStable := apitypes.GearFirmwareChannel(stable)
	putConfigResp, err := server.PutPeerConfig(ctx, adminservice.PutPeerConfigRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
		Body: &adminservice.PutPeerConfigJSONRequestBody{
			Certifications: &[]apitypes.GearCertification{{
				Type:      apitypes.GearCertificationType("license"),
				Authority: apitypes.GearCertificationAuthority("ce"),
				Id:        certID,
			}},
			Firmware: &apitypes.FirmwareConfig{Channel: &adminStable},
		},
	})
	if err != nil {
		t.Fatalf("PutPeerConfig error: %v", err)
	}
	if _, ok := putConfigResp.(adminservice.PutPeerConfig200JSONResponse); !ok {
		t.Fatalf("PutPeerConfig response type = %T", putConfigResp)
	}

	getConfigResp, err := server.GetPeerConfig(ctx, adminservice.GetPeerConfigRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
	})
	if err != nil {
		t.Fatalf("GetPeerConfig error: %v", err)
	}
	cfg, ok := getConfigResp.(adminservice.GetPeerConfig200JSONResponse)
	if !ok {
		t.Fatalf("GetPeerConfig response type = %T", getConfigResp)
	}
	if cfg.Firmware == nil || cfg.Firmware.Channel == nil || *cfg.Firmware.Channel != stable {
		t.Fatalf("GetPeerConfig = %+v", cfg)
	}

	getInfoResp, err := server.GetPeerInfo(ctx, adminservice.GetPeerInfoRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
	})
	if err != nil {
		t.Fatalf("GetPeerInfo error: %v", err)
	}
	info, ok := getInfoResp.(adminservice.GetPeerInfo200JSONResponse)
	if !ok {
		t.Fatalf("GetPeerInfo response type = %T", getInfoResp)
	}
	if info.Hardware == nil || info.Hardware.Imeis == nil || len(*info.Hardware.Imeis) != 1 {
		t.Fatalf("GetPeerInfo = %+v", info)
	}

	byFirmwareResp, err := server.ListPeersByFirmware(ctx, adminservice.ListPeersByFirmwareRequestObject{
		Depot:   depot,
		Channel: apitypes.GearFirmwareChannel(stable),
	})
	if err != nil {
		t.Fatalf("ListPeersByFirmware error: %v", err)
	}
	byFirmware, ok := byFirmwareResp.(adminservice.ListPeersByFirmware200JSONResponse)
	if !ok {
		t.Fatalf("ListPeersByFirmware response type = %T", byFirmwareResp)
	}
	if len(byFirmware.Items) != 1 || byFirmware.Items[0].PublicKey != peerPublicKey {
		t.Fatalf("ListPeersByFirmware items = %+v", byFirmware.Items)
	}

	resolveSNResp, err := server.ResolvePeerBySN(ctx, adminservice.ResolvePeerBySNRequestObject{Sn: sn})
	if err != nil {
		t.Fatalf("ResolvePeerBySN error: %v", err)
	}
	resolvedSN, ok := resolveSNResp.(adminservice.ResolvePeerBySN200JSONResponse)
	if !ok {
		t.Fatalf("ResolvePeerBySN response type = %T", resolveSNResp)
	}
	if resolvedSN.PublicKey != peerPublicKey {
		t.Fatalf("ResolvePeerBySN = %+v", resolvedSN)
	}

	byLabelResp, err := server.ListPeersByLabel(ctx, adminservice.ListPeersByLabelRequestObject{Key: labelKey, Value: labelValue})
	if err != nil {
		t.Fatalf("ListPeersByLabel error: %v", err)
	}
	byLabel, ok := byLabelResp.(adminservice.ListPeersByLabel200JSONResponse)
	if !ok {
		t.Fatalf("ListPeersByLabel response type = %T", byLabelResp)
	}
	if len(byLabel.Items) != 1 || byLabel.Items[0].PublicKey != peerPublicKey {
		t.Fatalf("ListPeersByLabel items = %+v", byLabel.Items)
	}

	resolveIMEIResp, err := server.ResolvePeerByIMEI(ctx, adminservice.ResolvePeerByIMEIRequestObject{
		Tac:    tac,
		Serial: serial,
	})
	if err != nil {
		t.Fatalf("ResolvePeerByIMEI error: %v", err)
	}
	resolvedIMEI, ok := resolveIMEIResp.(adminservice.ResolvePeerByIMEI200JSONResponse)
	if !ok {
		t.Fatalf("ResolvePeerByIMEI response type = %T", resolveIMEIResp)
	}
	if resolvedIMEI.PublicKey != peerPublicKey {
		t.Fatalf("ResolvePeerByIMEI = %+v", resolvedIMEI)
	}

	byCertificationResp, err := server.ListPeersByCertification(ctx, adminservice.ListPeersByCertificationRequestObject{
		Type:      apitypes.GearCertificationType("license"),
		Authority: apitypes.GearCertificationAuthority("ce"),
		Id:        certID,
	})
	if err != nil {
		t.Fatalf("ListPeersByCertification error: %v", err)
	}
	byCertification, ok := byCertificationResp.(adminservice.ListPeersByCertification200JSONResponse)
	if !ok {
		t.Fatalf("ListPeersByCertification response type = %T", byCertificationResp)
	}
	if len(byCertification.Items) != 1 || byCertification.Items[0].PublicKey != peerPublicKey {
		t.Fatalf("ListPeersByCertification items = %+v", byCertification.Items)
	}

	approveResp, err := server.ApprovePeer(ctx, adminservice.ApprovePeerRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
		Body:      &adminservice.ApprovePeerJSONRequestBody{Role: apitypes.GearRoleGear},
	})
	if err != nil {
		t.Fatalf("ApprovePeer error: %v", err)
	}
	approved, ok := approveResp.(adminservice.ApprovePeer200JSONResponse)
	if !ok {
		t.Fatalf("ApprovePeer response type = %T", approveResp)
	}
	if approved.Role != apitypes.GearRoleGear || approved.Status != apitypes.GearStatusActive {
		t.Fatalf("ApprovePeer = %+v", approved)
	}

	blockResp, err := server.BlockPeer(ctx, adminservice.BlockPeerRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
	})
	if err != nil {
		t.Fatalf("BlockPeer error: %v", err)
	}
	blocked, ok := blockResp.(adminservice.BlockPeer200JSONResponse)
	if !ok {
		t.Fatalf("BlockPeer response type = %T", blockResp)
	}
	if blocked.Status != apitypes.GearStatusBlocked {
		t.Fatalf("BlockPeer = %+v", blocked)
	}

	deleteResp, err := server.DeletePeer(ctx, adminservice.DeletePeerRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
	})
	if err != nil {
		t.Fatalf("DeletePeer error: %v", err)
	}
	deleted, ok := deleteResp.(adminservice.DeletePeer200JSONResponse)
	if !ok {
		t.Fatalf("DeletePeer response type = %T", deleteResp)
	}
	if deleted.Role != apitypes.GearRoleGear || deleted.Status != apitypes.GearStatusBlocked || deleted.ApprovedAt == nil {
		t.Fatalf("DeletePeer = %+v", deleted)
	}
	getDeletedResp, err := server.GetPeer(ctx, adminservice.GetPeerRequestObject{PublicKey: adminservice.PublicKey(peerPublicKey)})
	if err != nil {
		t.Fatalf("GetPeer after DeletePeer error: %v", err)
	}
	if _, ok := getDeletedResp.(adminservice.GetPeer404JSONResponse); !ok {
		t.Fatalf("GetPeer after DeletePeer response type = %T", getDeletedResp)
	}
}

func TestServerListPeersPagination(t *testing.T) {
	server := &Server{
		Store: mustBadgerInMemory(t, nil),
	}

	gearA := giznet.PublicKey{1}
	gearB := giznet.PublicKey{2}
	gearC := giznet.PublicKey{3}
	gearAText := gearA.String()
	gearBText := gearB.String()

	registerGear := func(publicKey giznet.PublicKey, labelValue string) {
		ctx := gearservice.WithCallerPublicKey(context.Background(), publicKey)
		_, err := server.RegisterGear(ctx, gearservice.RegisterGearRequestObject{
			Body: &gearservice.RegisterGearJSONRequestBody{
				Device: apitypes.DeviceInfo{
					Hardware: &apitypes.HardwareInfo{
						Labels: &[]apitypes.GearLabel{{Key: "region", Value: labelValue}},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("RegisterGear(%s) error: %v", publicKey, err)
		}
	}

	registerGear(gearA, "cn")
	registerGear(gearB, "cn")
	registerGear(gearC, "us")

	limit := adminservice.Limit(1)
	resp, err := server.ListPeers(context.Background(), adminservice.ListPeersRequestObject{
		Params: adminservice.ListPeersParams{
			Limit: &limit,
		},
	})
	if err != nil {
		t.Fatalf("ListPeers pagination error: %v", err)
	}
	listed, ok := resp.(adminservice.ListPeers200JSONResponse)
	if !ok {
		t.Fatalf("ListPeers response type = %T", resp)
	}
	if !listed.HasNext || listed.NextCursor == nil || *listed.NextCursor != gearAText {
		t.Fatalf("ListPeers pagination metadata = %+v", listed)
	}
	if len(listed.Items) != 1 || listed.Items[0].PublicKey != gearAText {
		t.Fatalf("ListPeers paged items = %+v", listed.Items)
	}

	firstFilteredResp, err := server.ListPeersByLabel(context.Background(), adminservice.ListPeersByLabelRequestObject{
		Key:   "region",
		Value: "cn",
		Params: adminservice.ListPeersByLabelParams{
			Limit: &limit,
		},
	})
	if err != nil {
		t.Fatalf("ListPeersByLabel pagination error: %v", err)
	}
	firstFiltered, ok := firstFilteredResp.(adminservice.ListPeersByLabel200JSONResponse)
	if !ok {
		t.Fatalf("ListPeersByLabel response type = %T", firstFilteredResp)
	}
	if !firstFiltered.HasNext || firstFiltered.NextCursor == nil || *firstFiltered.NextCursor != gearAText {
		t.Fatalf("ListPeersByLabel first page metadata = %+v", firstFiltered)
	}

	filteredResp, err := server.ListPeersByLabel(context.Background(), adminservice.ListPeersByLabelRequestObject{
		Key:   "region",
		Value: "cn",
		Params: adminservice.ListPeersByLabelParams{
			Cursor: firstFiltered.NextCursor,
			Limit:  &limit,
		},
	})
	if err != nil {
		t.Fatalf("ListPeersByLabel second page error: %v", err)
	}
	filtered, ok := filteredResp.(adminservice.ListPeersByLabel200JSONResponse)
	if !ok {
		t.Fatalf("ListPeersByLabel second response type = %T", filteredResp)
	}
	if filtered.HasNext || filtered.NextCursor != nil {
		t.Fatalf("ListPeersByLabel pagination metadata = %+v", filtered)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].PublicKey != gearBText {
		t.Fatalf("ListPeersByLabel paged items = %+v", filtered.Items)
	}
}

func TestServerListPeersPaginationPreservesCreationOrder(t *testing.T) {
	server := &Server{
		Store: mustBadgerInMemory(t, nil),
	}

	gearA := giznet.PublicKey{1}
	gearB := giznet.PublicKey{2}
	gearC := giznet.PublicKey{3}
	gearAText := gearA.String()
	gearBText := gearB.String()
	gearCText := gearC.String()

	registerGear := func(publicKey giznet.PublicKey) {
		ctx := gearservice.WithCallerPublicKey(context.Background(), publicKey)
		_, err := server.RegisterGear(ctx, gearservice.RegisterGearRequestObject{
			Body: &gearservice.RegisterGearJSONRequestBody{
				Device: apitypes.DeviceInfo{},
			},
		})
		if err != nil {
			t.Fatalf("RegisterGear(%s) error: %v", publicKey, err)
		}
	}

	registerGear(gearB)
	registerGear(gearA)
	registerGear(gearC)

	limit := adminservice.Limit(2)
	resp, err := server.ListPeers(context.Background(), adminservice.ListPeersRequestObject{
		Params: adminservice.ListPeersParams{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListPeers first page error: %v", err)
	}
	firstPage, ok := resp.(adminservice.ListPeers200JSONResponse)
	if !ok {
		t.Fatalf("ListPeers first response type = %T", resp)
	}
	if len(firstPage.Items) != 2 || firstPage.Items[0].PublicKey != gearBText || firstPage.Items[1].PublicKey != gearAText {
		t.Fatalf("ListPeers first page = %+v", firstPage.Items)
	}
	if !firstPage.HasNext || firstPage.NextCursor == nil || *firstPage.NextCursor != gearAText {
		t.Fatalf("ListPeers first page metadata = %+v", firstPage)
	}

	resp, err = server.ListPeers(context.Background(), adminservice.ListPeersRequestObject{
		Params: adminservice.ListPeersParams{
			Cursor: firstPage.NextCursor,
			Limit:  &limit,
		},
	})
	if err != nil {
		t.Fatalf("ListPeers second page error: %v", err)
	}
	secondPage, ok := resp.(adminservice.ListPeers200JSONResponse)
	if !ok {
		t.Fatalf("ListPeers second response type = %T", resp)
	}
	if len(secondPage.Items) != 1 || secondPage.Items[0].PublicKey != gearCText {
		t.Fatalf("ListPeers second page = %+v", secondPage.Items)
	}
}

func TestServerListPeersLimitClampsToConfiguredBounds(t *testing.T) {
	server := &Server{
		Store: mustBadgerInMemory(t, nil),
	}
	for _, publicKey := range []giznet.PublicKey{{1}, {2}, {3}} {
		ctx := gearservice.WithCallerPublicKey(context.Background(), publicKey)
		_, err := server.RegisterGear(ctx, gearservice.RegisterGearRequestObject{
			Body: &gearservice.RegisterGearJSONRequestBody{Device: apitypes.DeviceInfo{}},
		})
		if err != nil {
			t.Fatalf("RegisterGear(%s) error: %v", publicKey, err)
		}
	}

	zero := adminservice.Limit(0)
	resp, err := server.ListPeers(context.Background(), adminservice.ListPeersRequestObject{
		Params: adminservice.ListPeersParams{Limit: &zero},
	})
	if err != nil {
		t.Fatalf("ListPeers zero limit error: %v", err)
	}
	defaultPage, ok := resp.(adminservice.ListPeers200JSONResponse)
	if !ok {
		t.Fatalf("ListPeers zero limit response type = %T", resp)
	}
	if len(defaultPage.Items) != 3 || defaultPage.HasNext {
		t.Fatalf("ListPeers zero limit = %+v", defaultPage)
	}

	tooLarge := adminservice.Limit(999)
	resp, err = server.ListPeers(context.Background(), adminservice.ListPeersRequestObject{
		Params: adminservice.ListPeersParams{Limit: &tooLarge},
	})
	if err != nil {
		t.Fatalf("ListPeers large limit error: %v", err)
	}
	clampedPage, ok := resp.(adminservice.ListPeers200JSONResponse)
	if !ok {
		t.Fatalf("ListPeers large limit response type = %T", resp)
	}
	if len(clampedPage.Items) != 3 || clampedPage.HasNext {
		t.Fatalf("ListPeers large limit = %+v", clampedPage)
	}
}

func TestServerRuntimeHandlers(t *testing.T) {
	now := time.Unix(1_700_200_000, 0).UTC()
	runtimeAddr := "10.0.0.1:1234"
	peerKey := giznet.PublicKey{3}
	peerPublicKey := peerKey.String()
	server := &Server{
		Store: mustBadgerInMemory(t, nil),
		PeerManager: stubPeerManager{
			runtime: apitypes.Runtime{
				LastAddr:   &runtimeAddr,
				LastSeenAt: now,
				Online:     true,
			},
			refreshResult: adminservice.RefreshResult{
				Gear: apitypes.Gear{
					PublicKey: peerPublicKey,
					Role:      apitypes.GearRoleServer,
					Status:    apitypes.GearStatusActive,
				},
				UpdatedFields: &[]string{"device.name"},
			},
			refreshOnline: true,
		},
	}

	registerCtx := gearservice.WithCallerPublicKey(context.Background(), peerKey)
	gearCtx := gearservice.WithCallerPublicKey(context.Background(), peerKey)
	_, err := server.RegisterGear(registerCtx, gearservice.RegisterGearRequestObject{
		Body: &gearservice.RegisterGearJSONRequestBody{Device: apitypes.DeviceInfo{}},
	})
	if err != nil {
		t.Fatalf("RegisterGear error: %v", err)
	}

	getGearRuntimeResp, err := server.GetPeerRuntime(context.Background(), adminservice.GetPeerRuntimeRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
	})
	if err != nil {
		t.Fatalf("GetPeerRuntime error: %v", err)
	}
	gearRuntime, ok := getGearRuntimeResp.(adminservice.GetPeerRuntime200JSONResponse)
	if !ok {
		t.Fatalf("GetPeerRuntime response type = %T", getGearRuntimeResp)
	}
	if !gearRuntime.Online || gearRuntime.LastAddr == nil || *gearRuntime.LastAddr != runtimeAddr {
		t.Fatalf("GetPeerRuntime = %+v", gearRuntime)
	}

	getRuntimeResp, err := server.GetRuntime(gearCtx, gearservice.GetRuntimeRequestObject{})
	if err != nil {
		t.Fatalf("GetRuntime error: %v", err)
	}
	publicRuntime, ok := getRuntimeResp.(gearservice.GetRuntime200JSONResponse)
	if !ok {
		t.Fatalf("GetRuntime response type = %T", getRuntimeResp)
	}
	if !publicRuntime.Online || publicRuntime.LastAddr == nil || *publicRuntime.LastAddr != runtimeAddr {
		t.Fatalf("GetRuntime = %+v", publicRuntime)
	}

	refreshResp, err := server.RefreshPeer(context.Background(), adminservice.RefreshPeerRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
	})
	if err != nil {
		t.Fatalf("RefreshPeer error: %v", err)
	}
	refreshed, ok := refreshResp.(adminservice.RefreshPeer200JSONResponse)
	if !ok {
		t.Fatalf("RefreshPeer response type = %T", refreshResp)
	}
	if refreshed.Gear.PublicKey != peerPublicKey || refreshed.UpdatedFields == nil || len(*refreshed.UpdatedFields) != 1 {
		t.Fatalf("RefreshPeer = %+v", refreshed)
	}
}

func TestServerPublicHandlers(t *testing.T) {
	before := time.Now()
	peerKey := giznet.PublicKey{5}
	peerPublicKey := peerKey.String()
	server := &Server{
		Store:           mustBadgerInMemory(t, nil),
		BuildCommit:     "deadbeef",
		ServerPublicKey: giznet.PublicKey{1},
	}

	registerCtx := gearservice.WithCallerPublicKey(context.Background(), peerKey)
	gearCtx := gearservice.WithCallerPublicKey(context.Background(), peerKey)
	name := "gear-a"
	sn := "sn-1"
	depot := "alpha"
	labelKey := "region"
	labelValue := "cn"

	registerResp, err := server.RegisterGear(registerCtx, gearservice.RegisterGearRequestObject{
		Body: &gearservice.RegisterGearJSONRequestBody{
			Device: apitypes.DeviceInfo{
				Name: &name,
				Sn:   &sn,
				Hardware: &apitypes.HardwareInfo{
					Depot:  &depot,
					Labels: &[]apitypes.GearLabel{{Key: labelKey, Value: labelValue}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterGear error: %v", err)
	}

	registered, ok := registerResp.(gearservice.RegisterGear200JSONResponse)
	if !ok {
		t.Fatalf("RegisterGear response type = %T", registerResp)
	}
	if registered.Registration.PublicKey != peerPublicKey {
		t.Fatalf("PublicKey = %q", registered.Registration.PublicKey)
	}
	if registered.Registration.Role != apitypes.GearRole(apitypes.GearRoleUnspecified) {
		t.Fatalf("Role = %q", registered.Registration.Role)
	}
	if registered.Registration.Status != apitypes.GearStatus(apitypes.GearStatusActive) {
		t.Fatalf("Status = %q", registered.Registration.Status)
	}
	if registered.Registration.ApprovedAt != nil {
		t.Fatalf("ApprovedAt = %v", registered.Registration.ApprovedAt)
	}

	getInfoResp, err := server.GetInfo(gearCtx, gearservice.GetInfoRequestObject{})
	if err != nil {
		t.Fatalf("GetInfo error: %v", err)
	}
	info, ok := getInfoResp.(gearservice.GetInfo200JSONResponse)
	if !ok {
		t.Fatalf("GetInfo response type = %T", getInfoResp)
	}
	if info.Sn == nil || *info.Sn != sn {
		t.Fatalf("GetInfo sn = %v", info.Sn)
	}

	getRegistrationResp, err := server.GetRegistration(gearCtx, gearservice.GetRegistrationRequestObject{})
	if err != nil {
		t.Fatalf("GetRegistration error: %v", err)
	}
	publicRegistration, ok := getRegistrationResp.(gearservice.GetRegistration200JSONResponse)
	if !ok {
		t.Fatalf("GetRegistration response type = %T", getRegistrationResp)
	}
	if publicRegistration.Role != apitypes.GearRole(apitypes.GearRoleUnspecified) {
		t.Fatalf("GetRegistration role = %q", publicRegistration.Role)
	}

	serverInfoResp, err := server.GetServerInfo(registerCtx, serverpublic.GetServerInfoRequestObject{})
	if err != nil {
		t.Fatalf("GetServerInfo error: %v", err)
	}
	serverInfo, ok := serverInfoResp.(serverpublic.GetServerInfo200JSONResponse)
	if !ok {
		t.Fatalf("GetServerInfo response type = %T", serverInfoResp)
	}
	if serverInfo.BuildCommit != "deadbeef" || serverInfo.PublicKey != server.ServerPublicKey.String() {
		t.Fatalf("GetServerInfo = %+v", serverInfo)
	}
	if serverInfo.ServerTime < before.UnixMilli() || serverInfo.ServerTime > time.Now().Add(time.Second).UnixMilli() {
		t.Fatalf("GetServerInfo = %+v", serverInfo)
	}
}

func TestServerPublicHandlersPutInfoConfigAndRuntime(t *testing.T) {
	now := time.Unix(1_700_500_000, 0).UTC()
	runtimeAddr := "10.0.0.1:8888"
	peerKey := giznet.PublicKey{4}
	peerPublicKey := peerKey.String()
	server := &Server{
		Store: mustBadgerInMemory(t, nil),
		PeerManager: stubPeerManager{
			runtime: apitypes.Runtime{
				LastAddr:   &runtimeAddr,
				LastSeenAt: now,
				Online:     true,
			},
		},
	}

	registerCtx := gearservice.WithCallerPublicKey(context.Background(), peerKey)
	gearCtx := gearservice.WithCallerPublicKey(context.Background(), peerKey)
	sn := "sn-old"
	depot := "depot-public"
	_, err := server.RegisterGear(registerCtx, gearservice.RegisterGearRequestObject{
		Body: &gearservice.RegisterGearJSONRequestBody{
			Device: apitypes.DeviceInfo{
				Sn: &sn,
				Hardware: &apitypes.HardwareInfo{
					Depot: &depot,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterGear error: %v", err)
	}

	stable := apitypes.GearFirmwareChannel("stable")
	adminStable := apitypes.GearFirmwareChannel(stable)
	_, err = server.PutPeerConfig(context.Background(), adminservice.PutPeerConfigRequestObject{
		PublicKey: adminservice.PublicKey(peerPublicKey),
		Body: &adminservice.PutPeerConfigJSONRequestBody{
			Firmware: &apitypes.FirmwareConfig{Channel: &adminStable},
		},
	})
	if err != nil {
		t.Fatalf("PutPeerConfig error: %v", err)
	}

	getConfigResp, err := server.GetConfig(gearCtx, gearservice.GetConfigRequestObject{})
	if err != nil {
		t.Fatalf("GetConfig error: %v", err)
	}
	cfg, ok := getConfigResp.(gearservice.GetConfig200JSONResponse)
	if !ok {
		t.Fatalf("GetConfig response type = %T", getConfigResp)
	}
	if cfg.Firmware == nil || cfg.Firmware.Channel == nil || *cfg.Firmware.Channel != stable {
		t.Fatalf("GetConfig = %+v", cfg)
	}

	newSN := "sn-new"
	putInfoResp, err := server.PutInfo(gearCtx, gearservice.PutInfoRequestObject{
		Body: &gearservice.PutInfoJSONRequestBody{
			Sn: &newSN,
			Hardware: &apitypes.HardwareInfo{
				Depot: &depot,
			},
		},
	})
	if err != nil {
		t.Fatalf("PutInfo error: %v", err)
	}
	putInfo, ok := putInfoResp.(gearservice.PutInfo200JSONResponse)
	if !ok {
		t.Fatalf("PutInfo response type = %T", putInfoResp)
	}
	if putInfo.Sn == nil || *putInfo.Sn != newSN {
		t.Fatalf("PutInfo = %+v", putInfo)
	}

	getRuntimeResp, err := server.GetRuntime(gearCtx, gearservice.GetRuntimeRequestObject{})
	if err != nil {
		t.Fatalf("GetRuntime error: %v", err)
	}
	publicRuntime, ok := getRuntimeResp.(gearservice.GetRuntime200JSONResponse)
	if !ok {
		t.Fatalf("GetRuntime response type = %T", getRuntimeResp)
	}
	if !publicRuntime.Online || publicRuntime.LastAddr == nil || *publicRuntime.LastAddr != runtimeAddr {
		t.Fatalf("GetRuntime = %+v", publicRuntime)
	}
}
