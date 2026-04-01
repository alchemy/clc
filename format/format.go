package format

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Format defines the interface for a configuration file codec.
type Format interface {
	Decode(r io.Reader) (map[string]any, error)
	Encode(w io.Writer, data map[string]any) error
}

var (
	formats    = map[string]Format{}
	extensions = map[string]string{}
)

func register(name string, f Format, exts ...string) {
	formats[name] = f
	for _, ext := range exts {
		extensions[ext] = name
	}
}

func init() {
	register("json", JSON{}, ".json")
	register("json5", JSON5{}, ".json5")
	register("yaml", YAML{}, ".yaml", ".yml")
	register("toml", TOML{}, ".toml")
	register("ini", INI{}, ".ini", ".cfg", ".conf")
}

// Get returns the Format registered under the given name.
func Get(name string) (Format, error) {
	f, ok := formats[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown format: %q", name)
	}
	return f, nil
}

// Detect returns the format name for a file based on its extension.
func Detect(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	name, ok := extensions[ext]
	if !ok {
		return "", fmt.Errorf("cannot detect format from extension %q", ext)
	}
	return name, nil
}

// Names returns all registered format names.
func Names() []string {
	out := make([]string, 0, len(formats))
	for name := range formats {
		out = append(out, name)
	}
	return out
}
