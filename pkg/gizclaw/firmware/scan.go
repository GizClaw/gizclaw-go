package firmware

import (
	"context"
	"fmt"
	"sort"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func (s *Server) scanDepotNames(ctx context.Context) ([]string, error) {
	depots, err := s.listDepotMetadata(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(depots))
	for _, depot := range depots {
		if err := validateDepotName(depot.Name); err != nil {
			return nil, err
		}
		names = append(names, depot.Name)
	}
	sort.Strings(names)
	return names, nil
}

func (s *Server) scanDepot(ctx context.Context, name string) (apitypes.Depot, error) {
	depot, err := s.getDepotMetadata(ctx, name)
	if err != nil {
		return apitypes.Depot{}, err
	}
	if err := validateVersionOrder(depot); err != nil {
		return apitypes.Depot{}, err
	}
	return depot, nil
}

func (s *Server) scanRelease(ctx context.Context, depot string, channel Channel) (apitypes.DepotRelease, error) {
	snapshot, err := s.scanDepot(ctx, depot)
	if err != nil {
		return apitypes.DepotRelease{}, err
	}
	release, ok := depotRelease(snapshot, channel)
	if !ok {
		return apitypes.DepotRelease{}, errChannelNotFound
	}
	if err := validateReleaseAgainstFiles(s.store(), s.channelPath(depot, string(channel)), release); err != nil {
		return apitypes.DepotRelease{}, err
	}
	return normalizeDepotRelease(release), nil
}

func (s *Server) resolveOTA(ctx context.Context, depotName string, channel Channel) (apitypes.OTASummary, error) {
	depot, err := s.scanDepot(ctx, depotName)
	if err != nil {
		return apitypes.OTASummary{}, err
	}
	release, ok := depotRelease(depot, channel)
	if !ok {
		return apitypes.OTASummary{}, errFirmwareNotFound
	}
	if err := validateReleaseAgainstFiles(s.store(), s.channelPath(depotName, string(channel)), release); err != nil {
		return apitypes.OTASummary{}, fmt.Errorf("%w: %v", errFirmwareNotFound, err)
	}
	files := make([]apitypes.DepotFile, 0, len(releaseFiles(release)))
	for _, file := range releaseFiles(release) {
		files = append(files, apitypes.DepotFile{
			Md5:    file.Md5,
			Path:   file.Path,
			Sha256: file.Sha256,
		})
	}
	return apitypes.OTASummary{
		Channel:        string(channel),
		Depot:          depotName,
		Files:          files,
		FirmwareSemver: release.FirmwareSemver,
	}, nil
}
