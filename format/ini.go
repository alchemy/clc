package format

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/ini.v1"
)

type INI struct{}

func (INI) Decode(r io.Reader) (map[string]any, error) {
	f, err := ini.Load(io.NopCloser(r))
	if err != nil {
		return nil, err
	}

	data := make(map[string]any)

	// Default section keys go to top level.
	for _, key := range f.Section("").Keys() {
		data[key.Name()] = key.String()
	}

	// Named sections become nested maps.
	for _, sec := range f.Sections() {
		if sec.Name() == ini.DefaultSection {
			continue
		}
		m := make(map[string]any)
		for _, key := range sec.Keys() {
			m[key.Name()] = key.String()
		}
		data[sec.Name()] = m
	}

	return data, nil
}

func (INI) Encode(w io.Writer, data map[string]any) error {
	f := ini.Empty()

	for key, val := range data {
		switch v := val.(type) {
		case map[string]any:
			sec, err := f.NewSection(key)
			if err != nil {
				return err
			}
			for k, sv := range v {
				inner, ok := sv.(map[string]any)
				if ok {
					_ = inner
					return fmt.Errorf("ini: cannot encode nested section %s.%s (INI supports only one level of nesting)", key, k)
				}
				if _, err := sec.NewKey(k, toString(sv)); err != nil {
					return err
				}
			}
		default:
			if _, err := f.Section("").NewKey(key, toString(v)); err != nil {
				return err
			}
		}
	}

	_, err := f.WriteTo(w)
	return err
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = toString(item)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", val)
	}
}
