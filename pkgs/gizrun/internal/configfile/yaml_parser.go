package configfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

type YamlParser struct {
	parsers map[string]Parser
}

var _ Parser = (*YamlParser)(nil)

func NewYamlParser() *YamlParser {
	return &YamlParser{parsers: map[string]Parser{}}
}

func (p *YamlParser) ParseFile(path string) (ConfigFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return ConfigFile{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ConfigFile{}, err
	}
	value, err := p.Parse(data)
	if err != nil {
		return ConfigFile{}, fmt.Errorf("parse %s: %w", path, err)
	}
	config, ok := value.(ConfigFile)
	if !ok {
		return ConfigFile{}, fmt.Errorf("parse %s: expected config file, got %T", path, value)
	}
	return config, nil
}

func (p *YamlParser) Parse(data []byte) (any, error) {
	if len(data) == 0 {
		return ConfigFile{}, nil
	}
	var sections map[string]yaml.RawMessage
	if err := yaml.Unmarshal(data, &sections); err != nil {
		return ConfigFile{}, err
	}
	config := make(ConfigFile, len(sections))
	for name, value := range sections {
		parser, ok := p.parser(name)
		if !ok {
			config[name] = value
			continue
		}
		section, err := parser.Parse(value)
		if err != nil {
			return ConfigFile{}, fmt.Errorf("%s: %w", name, err)
		}
		config[name] = section
	}
	return config, nil
}

func (p *YamlParser) Register(name string, parser Parser) {
	name = strings.TrimSpace(name)
	if name == "" || parser == nil {
		return
	}
	if p.parsers == nil {
		p.parsers = map[string]Parser{}
	}
	p.parsers[name] = parser
}

func (p *YamlParser) parser(name string) (Parser, bool) {
	if p == nil || p.parsers == nil {
		return nil, false
	}
	parser, ok := p.parsers[name]
	return parser, ok
}
