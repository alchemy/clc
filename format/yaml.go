package format

import (
	"io"

	"gopkg.in/yaml.v3"
)

type YAML struct{}

func (YAML) Decode(r io.Reader) (map[string]any, error) {
	var data map[string]any
	if err := yaml.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (YAML) Encode(w io.Writer, data map[string]any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(data)
}
