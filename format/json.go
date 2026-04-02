package format

import (
	"encoding/json"
	"io"
)

type JSON struct{}

func (JSON) Decode(r io.Reader) (*Document, error) {
	var data map[string]any
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return &Document{
		Data:     data,
		Comments: make(map[string]Comment),
		KeyOrder: make(map[string][]string),
	}, nil
}

func (JSON) Encode(w io.Writer, doc *Document) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc.Data)
}
