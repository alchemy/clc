package format

import "sort"

// Comment holds comment text associated with a configuration key.
// Text is stored without format-specific prefixes (no #, //, etc.).
type Comment struct {
	Head   string // Comment block above the key
	Inline string // Comment on the same line as the value
}

// Document holds configuration data along with associated comments
// and key ordering metadata.
type Document struct {
	Data     map[string]any
	Comments map[string]Comment   // key path (e.g. "server.port") -> comment
	KeyOrder map[string][]string  // path prefix -> ordered keys at that level ("" = top-level)
	Header   string               // Comment at the top of the file, before any data
	Footer   string               // Comment at the bottom of the file, after all data
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// orderedKeysFor returns keys for the given data map, using the stored order
// from keyOrder[prefix] if available, falling back to sorted keys.
func orderedKeysFor(data map[string]any, keyOrder map[string][]string, prefix string) []string {
	order, ok := keyOrder[prefix]
	if !ok || len(order) == 0 {
		return sortedKeys(data)
	}

	seen := make(map[string]bool, len(order))
	result := make([]string, 0, len(data))
	for _, k := range order {
		if _, exists := data[k]; exists {
			result = append(result, k)
			seen[k] = true
		}
	}
	// Append any keys not in the stored order (sorted).
	var extra []string
	for k := range data {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return append(result, extra...)
}
