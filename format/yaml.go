package format

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type YAML struct{}

func (YAML) Decode(r io.Reader) (*Document, error) {
	var node yaml.Node
	if err := yaml.NewDecoder(r).Decode(&node); err != nil {
		if err == io.EOF {
			return &Document{
				Data:     make(map[string]any),
				Comments: make(map[string]Comment),
				KeyOrder: make(map[string][]string),
			}, nil
		}
		return nil, err
	}

	doc := &Document{
		Comments: make(map[string]Comment),
		KeyOrder: make(map[string][]string),
	}

	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		doc.Header = cleanYAMLComment(node.HeadComment)
		doc.Footer = cleanYAMLComment(node.FootComment)
		root := node.Content[0]
		if root.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("yaml: expected top-level mapping")
		}
		doc.Data = yamlMappingToData(root, "", doc)
	} else {
		doc.Data = make(map[string]any)
	}

	return doc, nil
}

func (YAML) Encode(w io.Writer, doc *Document) error {
	root := buildYAMLMappingNode(doc.Data, doc.Comments, doc.KeyOrder, "")

	docNode := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{root},
	}
	if doc.Header != "" {
		docNode.HeadComment = formatYAMLComment(doc.Header)
	}
	if doc.Footer != "" {
		docNode.FootComment = formatYAMLComment(doc.Footer)
	}

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(docNode)
}

// --- Decode helpers ---

func yamlMappingToData(node *yaml.Node, prefix string, doc *Document) map[string]any {
	data := make(map[string]any)
	keys := make([]string, 0, len(node.Content)/2)

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		key := keyNode.Value
		keys = append(keys, key)

		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		// Extract comments.
		c := Comment{}
		if keyNode.HeadComment != "" {
			c.Head = cleanYAMLComment(keyNode.HeadComment)
		}
		// Inline comment: prefer value node, fall back to key node.
		if valNode.LineComment != "" {
			c.Inline = cleanYAMLComment(valNode.LineComment)
		} else if keyNode.LineComment != "" {
			c.Inline = cleanYAMLComment(keyNode.LineComment)
		}
		if c.Head != "" || c.Inline != "" {
			doc.Comments[path] = c
		}

		data[key] = yamlNodeToValue(valNode, path, doc)
	}

	doc.KeyOrder[prefix] = keys
	return data
}

func yamlNodeToValue(node *yaml.Node, path string, doc *Document) any {
	switch node.Kind {
	case yaml.MappingNode:
		return yamlMappingToData(node, path, doc)
	case yaml.SequenceNode:
		return yamlSequenceToSlice(node, path, doc)
	case yaml.ScalarNode:
		return yamlScalarValue(node)
	case yaml.AliasNode:
		if node.Alias != nil {
			return yamlNodeToValue(node.Alias, path, doc)
		}
		return nil
	default:
		return nil
	}
}

func yamlSequenceToSlice(node *yaml.Node, prefix string, doc *Document) []any {
	result := make([]any, 0, len(node.Content))
	for i, item := range node.Content {
		itemPath := fmt.Sprintf("%s[%d]", prefix, i)

		c := Comment{}
		if item.HeadComment != "" {
			c.Head = cleanYAMLComment(item.HeadComment)
		}
		if item.LineComment != "" {
			c.Inline = cleanYAMLComment(item.LineComment)
		}
		if c.Head != "" || c.Inline != "" {
			doc.Comments[itemPath] = c
		}

		result = append(result, yamlNodeToValue(item, itemPath, doc))
	}
	return result
}

func yamlScalarValue(node *yaml.Node) any {
	switch node.Tag {
	case "!!bool":
		return node.Value == "true"
	case "!!int":
		s := strings.ReplaceAll(node.Value, "_", "")
		i, err := strconv.ParseInt(s, 0, 64)
		if err == nil {
			return i
		}
		return node.Value
	case "!!float":
		s := strings.ReplaceAll(node.Value, "_", "")
		switch strings.ToLower(s) {
		case ".inf", "+.inf":
			return math.Inf(1)
		case "-.inf":
			return math.Inf(-1)
		case ".nan":
			return math.NaN()
		}
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return f
		}
		return node.Value
	case "!!null":
		return nil
	default:
		return node.Value
	}
}

// --- Encode helpers ---

func buildYAMLMappingNode(data map[string]any, comments map[string]Comment, keyOrder map[string][]string, prefix string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}

	keys := orderedKeysFor(data, keyOrder, prefix)
	for _, key := range keys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
			Tag:   "!!str",
		}

		if c, ok := comments[path]; ok && c.Head != "" {
			keyNode.HeadComment = formatYAMLComment(c.Head)
		}

		valNode := buildYAMLValueNode(data[key], path, comments, keyOrder)

		if c, ok := comments[path]; ok && c.Inline != "" {
			if valNode.Kind == yaml.MappingNode || valNode.Kind == yaml.SequenceNode {
				keyNode.LineComment = formatYAMLComment(c.Inline)
			} else {
				valNode.LineComment = formatYAMLComment(c.Inline)
			}
		}

		node.Content = append(node.Content, keyNode, valNode)
	}

	return node
}

func buildYAMLValueNode(val any, path string, comments map[string]Comment, keyOrder map[string][]string) *yaml.Node {
	switch v := val.(type) {
	case map[string]any:
		return buildYAMLMappingNode(v, comments, keyOrder, path)
	case []any:
		return buildYAMLSequenceNode(v, path, comments, keyOrder)
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: "!!str"}
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatBool(v), Tag: "!!bool"}
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.Itoa(v), Tag: "!!int"}
	case int64:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatInt(v, 10), Tag: "!!int"}
	case float64:
		if math.IsInf(v, 1) {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: ".inf", Tag: "!!float"}
		}
		if math.IsInf(v, -1) {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: "-.inf", Tag: "!!float"}
		}
		if math.IsNaN(v) {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: ".nan", Tag: "!!float"}
		}
		if v == float64(int64(v)) && v >= math.MinInt64 && v <= math.MaxInt64 {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatInt(int64(v), 10), Tag: "!!int"}
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatFloat(v, 'f', -1, 64), Tag: "!!float"}
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null", Tag: "!!null"}
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", v)}
	}
}

func buildYAMLSequenceNode(arr []any, prefix string, comments map[string]Comment, keyOrder map[string][]string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.SequenceNode}
	for i, item := range arr {
		itemPath := fmt.Sprintf("%s[%d]", prefix, i)
		child := buildYAMLValueNode(item, itemPath, comments, keyOrder)
		if c, ok := comments[itemPath]; ok {
			if c.Head != "" {
				child.HeadComment = formatYAMLComment(c.Head)
			}
			if c.Inline != "" {
				child.LineComment = formatYAMLComment(c.Inline)
			}
		}
		node.Content = append(node.Content, child)
	}
	return node
}

// --- Comment formatting ---

func cleanYAMLComment(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "#")
		if len(line) > 0 && line[0] == ' ' {
			line = line[1:]
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func formatYAMLComment(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "# " + line
	}
	return strings.Join(lines, "\n")
}
