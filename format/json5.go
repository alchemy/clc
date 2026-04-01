package format

import (
	"io"

	"github.com/alchemy/json5"
)

type JSON5 struct{}

func (JSON5) Decode(r io.Reader) (map[string]any, error) {
	var data map[string]any
	if err := json5.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (JSON5) Encode(w io.Writer, data map[string]any) error {
	b, err := json5.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}
