package format

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/ini.v1"
)

type INI struct{}

func (INI) Decode(r io.Reader) (*Document, error) {
	f, err := ini.Load(io.NopCloser(r))
	if err != nil {
		return nil, err
	}

	doc := &Document{
		Data:     make(map[string]any),
		Comments: make(map[string]Comment),
		KeyOrder: make(map[string][]string),
	}

	// Default section keys go to top level.
	topKeys := make([]string, 0)
	for _, key := range f.Section("").Keys() {
		doc.Data[key.Name()] = key.String()
		topKeys = append(topKeys, key.Name())
		if key.Comment != "" {
			doc.Comments[key.Name()] = Comment{Head: cleanINIComment(key.Comment)}
		}
	}

	// Named sections become nested maps.
	for _, sec := range f.Sections() {
		if sec.Name() == ini.DefaultSection {
			continue
		}
		topKeys = append(topKeys, sec.Name())

		m := make(map[string]any)
		secKeys := make([]string, 0)
		for _, key := range sec.Keys() {
			m[key.Name()] = key.String()
			secKeys = append(secKeys, key.Name())
			path := sec.Name() + "." + key.Name()
			if key.Comment != "" {
				doc.Comments[path] = Comment{Head: cleanINIComment(key.Comment)}
			}
		}
		doc.Data[sec.Name()] = m
		doc.KeyOrder[sec.Name()] = secKeys

		if sec.Comment != "" {
			doc.Comments[sec.Name()] = Comment{Head: cleanINIComment(sec.Comment)}
		}
	}

	doc.KeyOrder[""] = topKeys
	return doc, nil
}

func (INI) Encode(w io.Writer, doc *Document) error {
	f := ini.Empty()

	keys := orderedKeysFor(doc.Data, doc.KeyOrder, "")
	for _, key := range keys {
		val := doc.Data[key]
		switch v := val.(type) {
		case map[string]any:
			sec, err := f.NewSection(key)
			if err != nil {
				return err
			}
			if c, ok := doc.Comments[key]; ok && c.Head != "" {
				sec.Comment = formatINIComment(c.Head)
			}
			subKeys := orderedKeysFor(v, doc.KeyOrder, key)
			for _, k := range subKeys {
				sv := v[k]
				if _, nested := sv.(map[string]any); nested {
					return fmt.Errorf("ini: cannot encode nested section %s.%s (INI supports only one level of nesting)", key, k)
				}
				newKey, err := sec.NewKey(k, toString(sv))
				if err != nil {
					return err
				}
				path := key + "." + k
				if c, ok := doc.Comments[path]; ok && c.Head != "" {
					newKey.Comment = formatINIComment(c.Head)
				}
			}
		default:
			newKey, err := f.Section("").NewKey(key, toString(v))
			if err != nil {
				return err
			}
			if c, ok := doc.Comments[key]; ok && c.Head != "" {
				newKey.Comment = formatINIComment(c.Head)
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

func cleanINIComment(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimPrefix(line, ";")
		if len(line) > 0 && line[0] == ' ' {
			line = line[1:]
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func formatINIComment(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "# " + line
	}
	return strings.Join(lines, "\n")
}
