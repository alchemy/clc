package format

import (
	"fmt"
	"io"

	"github.com/alchemy/json5"
)

type JSON5 struct{}

func (JSON5) Decode(r io.Reader) (*Document, error) {
	dec := json5.NewDecoder(r)
	dec.UseOrderedMap()
	var raw any
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}

	comments := make(map[string]Comment)
	keyOrder := make(map[string][]string)
	data := convertOrderedValue(raw, "", keyOrder, comments)
	m, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("json5: expected top-level object")
	}

	return &Document{
		Data:     m,
		Comments: comments,
		KeyOrder: keyOrder,
	}, nil
}

func (JSON5) Encode(w io.Writer, doc *Document) error {
	ordered := buildOrderedMap(doc.Data, doc.Comments, doc.KeyOrder, "")
	b, err := json5.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

// convertOrderedValue recursively converts *json5.OrderedMap values to
// map[string]any while recording key ordering and extracting comments.
func convertOrderedValue(v any, prefix string, keyOrder map[string][]string, comments map[string]Comment) any {
	switch val := v.(type) {
	case *json5.OrderedMap:
		m := make(map[string]any)
		keys := make([]string, 0, val.Len())
		for _, entry := range val.Entries() {
			keys = append(keys, entry.Key)
			childPrefix := entry.Key
			if prefix != "" {
				childPrefix = prefix + "." + entry.Key
			}
			m[entry.Key] = convertOrderedValue(entry.Value, childPrefix, keyOrder, comments)

			// Extract comments from entry.
			if entry.Comment != "" || entry.InlineComment != "" {
				comments[childPrefix] = Comment{
					Head:   entry.Comment,
					Inline: entry.InlineComment,
				}
			}
		}
		keyOrder[prefix] = keys
		return m
	case []any:
		for i, item := range val {
			itemPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			val[i] = convertOrderedValue(item, itemPrefix, keyOrder, comments)
		}
		return val
	default:
		return v
	}
}

// buildOrderedMap reconstructs a *json5.OrderedMap from data + comments + keyOrder
// so that json5.MarshalIndent preserves key ordering and comments.
func buildOrderedMap(data map[string]any, comments map[string]Comment, keyOrder map[string][]string, prefix string) *json5.OrderedMap {
	om := json5.NewOrderedMap()
	keys := orderedKeysFor(data, keyOrder, prefix)
	for _, key := range keys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		val := data[key]
		if m, ok := val.(map[string]any); ok {
			val = buildOrderedMap(m, comments, keyOrder, path)
		}

		c := comments[path]
		om.SetWithComment(key, val, c.Head, c.Inline)
	}
	return om
}
