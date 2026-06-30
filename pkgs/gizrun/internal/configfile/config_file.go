package configfile

type Parser interface {
	Parse([]byte) (any, error)
}

type ParseFunc func([]byte) (any, error)

func (fn ParseFunc) Parse(data []byte) (any, error) {
	return fn(data)
}

type ConfigFile map[string]any

func (c ConfigFile) Config(name string) (any, bool) {
	if c == nil {
		return nil, false
	}
	value, ok := c[name]
	return value, ok
}

func (c ConfigFile) Len() int {
	return len(c)
}
