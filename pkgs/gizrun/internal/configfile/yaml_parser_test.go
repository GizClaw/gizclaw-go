package configfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestLoadReturnsRegisteredAndRawSections(t *testing.T) {
	registry := NewYamlParser()

	type logConfig struct {
		Level string `yaml:"level"`
	}
	type metricsConfig struct {
		Enabled bool `yaml:"enabled"`
	}
	registry.Register("log", ParseFunc(func(data []byte) (any, error) {
		var config logConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return config, nil
	}))
	registry.Register("metrics", ParseFunc(func(data []byte) (any, error) {
		var config metricsConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return config, nil
	}))

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
log:
  level: debug
metrics:
  enabled: true
unknown:
  ignored: true
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := registry.ParseFile(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	section, ok := config.Config("log")
	if !ok {
		t.Fatal("log section not found")
	}
	logSection, ok := section.(logConfig)
	if !ok {
		t.Fatalf("log section type = %T, want logConfig", section)
	}
	if logSection.Level != "debug" {
		t.Fatalf("log level = %q, want debug", logSection.Level)
	}
	section, ok = config.Config("metrics")
	if !ok {
		t.Fatal("metrics section not found")
	}
	metricsSection, ok := section.(metricsConfig)
	if !ok {
		t.Fatalf("metrics section type = %T, want metricsConfig", section)
	}
	if !metricsSection.Enabled {
		t.Fatal("metrics enabled = false, want true")
	}
	section, ok = config.Config("unknown")
	if !ok {
		t.Fatal("unknown section not found")
	}
	if _, ok := section.(yaml.RawMessage); !ok {
		t.Fatalf("unknown section type = %T, want yaml.RawMessage", section)
	}
}

func TestRegisterIgnoresInvalidParser(t *testing.T) {
	registry := NewYamlParser()

	registry.Register("", ParseFunc(func([]byte) (any, error) { return struct{}{}, nil }))
	registry.Register("valid", nil)

	value, err := registry.Parse([]byte("valid: {}\n"))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	config, ok := value.(ConfigFile)
	if !ok {
		t.Fatalf("Parse value type = %T, want ConfigFile", value)
	}
	section, ok := config.Config("valid")
	if !ok {
		t.Fatal("valid section not found")
	}
	if _, ok := section.(yaml.RawMessage); !ok {
		t.Fatalf("valid section type = %T, want yaml.RawMessage", section)
	}
}

func TestParseWrapsUnmarshalError(t *testing.T) {
	registry := NewYamlParser()

	registry.Register("log", ParseFunc(func(data []byte) (any, error) {
		var config yamlErrorParser
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return config, nil
	}))

	_, err := registry.Parse([]byte("log: {}\n"))
	if !errors.Is(err, errBadConfig) {
		t.Fatalf("Parse error = %v, want wrapping %v", err, errBadConfig)
	}
}

func TestLoadEmptyPathAndParseEmptyData(t *testing.T) {
	registry := NewYamlParser()

	config, err := registry.ParseFile("")
	if err != nil {
		t.Fatalf("Load empty path failed: %v", err)
	}
	if config.Len() != 0 {
		t.Fatalf("Load empty path returned %d sections, want 0", config.Len())
	}
	value, err := registry.Parse(nil)
	if err != nil {
		t.Fatalf("Parse empty data failed: %v", err)
	}
	config, ok := value.(ConfigFile)
	if !ok {
		t.Fatalf("Parse value type = %T, want ConfigFile", value)
	}
	if config.Len() != 0 {
		t.Fatalf("Parse empty data returned %d sections, want 0", config.Len())
	}
}

var errBadConfig = errors.New("bad config")

type yamlErrorParser struct{}

func (p *yamlErrorParser) UnmarshalYAML([]byte) error {
	return errBadConfig
}
