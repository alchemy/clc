package format

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type TOML struct{}

func (TOML) Decode(r io.Reader) (*Document, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	meta, err := toml.NewDecoder(bytes.NewReader(raw)).Decode(&data)
	if err != nil {
		return nil, err
	}

	comments, header, footer := extractTOMLComments(raw)
	keyOrder := buildTOMLKeyOrder(meta.Keys())

	return &Document{
		Data:     data,
		Comments: comments,
		KeyOrder: keyOrder,
		Header:   header,
		Footer:   footer,
	}, nil
}

func (TOML) Encode(w io.Writer, doc *Document) error {
	var buf bytes.Buffer

	if doc.Header != "" {
		writeTOMLCommentBlock(&buf, doc.Header)
		buf.WriteByte('\n')
	}

	writeTOMLSection(&buf, doc.Data, doc.Comments, doc.KeyOrder, "")

	if doc.Footer != "" {
		buf.WriteByte('\n')
		writeTOMLCommentBlock(&buf, doc.Footer)
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// --- Key ordering from MetaData ---

func buildTOMLKeyOrder(metaKeys []toml.Key) map[string][]string {
	order := make(map[string][]string)
	added := make(map[string]bool)

	for _, key := range metaKeys {
		if len(key) == 0 {
			continue
		}
		parts := []string(key)

		// Register intermediate sections with their parent levels.
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[:i-1], ".")
			section := parts[i-1]
			uid := parent + "\x00" + section
			if !added[uid] {
				added[uid] = true
				order[parent] = append(order[parent], section)
			}
		}

		// Register the leaf key with its parent.
		parent := strings.Join(parts[:len(parts)-1], ".")
		leaf := parts[len(parts)-1]
		uid := parent + "\x00" + leaf
		if !added[uid] {
			added[uid] = true
			order[parent] = append(order[parent], leaf)
		}
	}

	return order
}

// --- Comment extraction from raw TOML text ---

func extractTOMLComments(raw []byte) (comments map[string]Comment, header, footer string) {
	comments = make(map[string]Comment)
	lines := strings.Split(string(raw), "\n")

	var section string
	var pending []string
	contentSeen := false

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)

		// Empty line.
		if trimmed == "" {
			// Before any content, a blank line after comments promotes them to header.
			if !contentSeen && len(pending) > 0 {
				if header != "" {
					header += "\n" + strings.Join(pending, "\n")
				} else {
					header = strings.Join(pending, "\n")
				}
				pending = nil
			}
			continue
		}

		// Comment line.
		if trimmed[0] == '#' {
			text := strings.TrimLeft(trimmed, "#")
			text = strings.TrimSpace(text)
			pending = append(pending, text)
			continue
		}

		contentSeen = true

		// Section header: [name] or [[name]]
		if trimmed[0] == '[' {
			section = parseTOMLSectionName(trimmed)
			if len(pending) > 0 {
				c := comments[section]
				c.Head = strings.Join(pending, "\n")
				comments[section] = c
				pending = nil
			}
			continue
		}

		// Key = value line.
		eqIdx := findTOMLEquals(trimmed)
		if eqIdx > 0 {
			key := strings.TrimSpace(trimmed[:eqIdx])
			key = strings.Trim(key, `"'`)

			path := key
			if section != "" {
				path = section + "." + key
			}

			c := comments[path]
			if len(pending) > 0 {
				c.Head = strings.Join(pending, "\n")
				pending = nil
			}

			inline := extractTOMLInlineComment(trimmed[eqIdx+1:])
			if inline != "" {
				c.Inline = inline
			}

			if c.Head != "" || c.Inline != "" {
				comments[path] = c
			}
		}
	}

	if len(pending) > 0 {
		footer = strings.Join(pending, "\n")
	}

	return
}

func parseTOMLSectionName(line string) string {
	s := line
	// Strip [[ for array of tables.
	isArray := strings.HasPrefix(s, "[[")
	if isArray {
		s = s[2:]
	} else {
		s = s[1:]
	}
	end := strings.Index(s, "]")
	if end >= 0 {
		s = s[:end]
	}
	return strings.TrimSpace(s)
}

// findTOMLEquals returns the index of the first '=' that is not inside a quoted string.
func findTOMLEquals(line string) int {
	inString := false
	var quote byte
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inString {
			if ch == '\\' && quote == '"' {
				i++
				continue
			}
			if ch == quote {
				inString = false
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inString = true
			quote = ch
			continue
		}
		if ch == '=' {
			return i
		}
		if ch == '[' || ch == '#' {
			return -1
		}
	}
	return -1
}

// extractTOMLInlineComment finds a '#' comment after a TOML value, skipping '#' inside strings.
func extractTOMLInlineComment(valuePart string) string {
	s := strings.TrimSpace(valuePart)
	inString := false
	var quote byte
	tripleQuote := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if tripleQuote {
				delim := string([]byte{quote, quote, quote})
				if i+2 < len(s) && s[i:i+3] == delim {
					inString = false
					i += 2
				}
				continue
			}
			if ch == '\\' && quote == '"' {
				i++
				continue
			}
			if ch == quote {
				inString = false
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			if i+2 < len(s) && s[i+1] == ch && s[i+2] == ch {
				inString = true
				quote = ch
				tripleQuote = true
				i += 2
				continue
			}
			inString = true
			quote = ch
			tripleQuote = false
			continue
		}
		if ch == '#' {
			rest := s[i:]
			rest = strings.TrimLeft(rest, "#")
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// --- Custom TOML encoder ---

func writeTOMLSection(buf *bytes.Buffer, data map[string]any, comments map[string]Comment, keyOrder map[string][]string, sectionPath string) {
	keys := orderedKeysFor(data, keyOrder, sectionPath)

	// First pass: write scalar (non-map) values.
	for _, key := range keys {
		val := data[key]
		if _, isMap := val.(map[string]any); isMap {
			continue
		}

		path := key
		if sectionPath != "" {
			path = sectionPath + "." + key
		}

		if c, ok := comments[path]; ok && c.Head != "" {
			writeTOMLCommentBlock(buf, c.Head)
		}

		fmt.Fprintf(buf, "%s = %s", tomlQuoteKey(key), tomlFormatValue(val))

		if c, ok := comments[path]; ok && c.Inline != "" {
			fmt.Fprintf(buf, " # %s", c.Inline)
		}

		buf.WriteByte('\n')
	}

	// Second pass: write subsections (map values).
	for _, key := range keys {
		val := data[key]
		m, isMap := val.(map[string]any)
		if !isMap {
			continue
		}

		subPath := key
		if sectionPath != "" {
			subPath = sectionPath + "." + key
		}

		buf.WriteByte('\n')

		if c, ok := comments[subPath]; ok && c.Head != "" {
			writeTOMLCommentBlock(buf, c.Head)
		}

		fmt.Fprintf(buf, "[%s]\n", subPath)
		writeTOMLSection(buf, m, comments, keyOrder, subPath)
	}
}

func writeTOMLCommentBlock(buf *bytes.Buffer, comment string) {
	for _, line := range strings.Split(comment, "\n") {
		if line == "" {
			buf.WriteString("#\n")
		} else {
			fmt.Fprintf(buf, "# %s\n", line)
		}
	}
}

func tomlQuoteKey(key string) string {
	// Bare keys: only letters, digits, dashes, underscores.
	bare := true
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			bare = false
			break
		}
	}
	if bare && key != "" {
		return key
	}
	return strconv.Quote(key)
}

func tomlFormatValue(val any) string {
	switch v := val.(type) {
	case string:
		return strconv.Quote(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int64(v)) && !math.IsInf(v, 0) && !math.IsNaN(v) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = tomlFormatValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case nil:
		return `""`
	default:
		return fmt.Sprintf(`"%v"`, v)
	}
}
