package gizrun

import (
	"log/slog"
	"strings"
)

type logLevelFlag struct {
	target *slog.LevelVar
}

func (v logLevelFlag) String() string {
	if v.target == nil {
		return slog.LevelInfo.String()
	}
	return v.target.Level().String()
}

func (v logLevelFlag) Set(value string) error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(value))); err != nil {
		return err
	}
	v.target.Set(level)
	return nil
}
