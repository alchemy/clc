package format

import (
	"io"

	"github.com/BurntSushi/toml"
)

type TOML struct{}

func (TOML) Decode(r io.Reader) (map[string]any, error) {
	var data map[string]any
	if _, err := toml.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (TOML) Encode(w io.Writer, data map[string]any) error {
	return toml.NewEncoder(w).Encode(data)
}
