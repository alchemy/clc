package format

import (
	"encoding/json"
	"io"
)

type JSON struct{}

func (JSON) Decode(r io.Reader) (map[string]any, error) {
	var data map[string]any
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (JSON) Encode(w io.Writer, data map[string]any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
