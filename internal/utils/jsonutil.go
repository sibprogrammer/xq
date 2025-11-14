package utils

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/antchfx/xmlquery"
)

// OrderedMap preserves insertion order for JSON output
type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

// NewOrderedMap creates a new OrderedMap
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0),
		values: make(map[string]interface{}),
	}
}

// Set adds or updates a key-value pair
func (om *OrderedMap) Set(key string, value interface{}) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

// Get retrieves a value by key
func (om *OrderedMap) Get(key string) (interface{}, bool) {
	val, ok := om.values[key]
	return val, ok
}

// Len returns the number of entries
func (om *OrderedMap) Len() int {
	return len(om.keys)
}

// MarshalJSON implements json.Marshaler to preserve order
func (om *OrderedMap) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteByte('{')
	for i, key := range om.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		// Marshal the key
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')
		// Marshal the value
		valBytes, err := json.Marshal(om.values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// NodeToJSON converts an xmlquery.Node to a JSON object. The depth parameter
// specifies how many levels of children to include in the result. A depth of 0 means
// only the text content of the node is included. A depth of 1 means the node's children
// are included, but not their children, and so on.
func NodeToJSON(node *xmlquery.Node, depth int) interface{} {
	if node == nil {
		return nil
	}

	switch node.Type {
	case xmlquery.DocumentNode:
		result := NewOrderedMap()
		var textParts []string

		// Process the next sibling of the document node first (if any)
		if node.NextSibling != nil && (node.NextSibling.Type == xmlquery.TextNode || node.NextSibling.Type == xmlquery.CharDataNode) {
			text := strings.TrimSpace(node.NextSibling.Data)
			if text != "" {
				textParts = append(textParts, text)
			}
		}

		// Process all children, including siblings of the first child
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			switch child.Type {
			case xmlquery.ElementNode:
				childResult := nodeToJSONInternal(child, depth)
				result.Set(child.Data, childResult)
			case xmlquery.TextNode, xmlquery.CharDataNode:
				text := strings.TrimSpace(child.Data)
				if text != "" {
					textParts = append(textParts, text)
				}
			}
		}

		if len(textParts) > 0 {
			result.Set("#text", strings.Join(textParts, "\n"))
		}
		return result

	case xmlquery.ElementNode:
		return nodeToJSONInternal(node, depth)

	case xmlquery.TextNode, xmlquery.CharDataNode:
		return strings.TrimSpace(node.Data)

	default:
		return nil
	}
}

func nodeToJSONInternal(node *xmlquery.Node, depth int) interface{} {
	if depth == 0 {
		return getTextContent(node)
	}

	result := NewOrderedMap()
	for _, attr := range node.Attr {
		result.Set("@"+attr.Name.Local, attr.Value)
	}

	var textParts []string
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case xmlquery.TextNode, xmlquery.CharDataNode:
			text := strings.TrimSpace(child.Data)
			if text != "" {
				textParts = append(textParts, text)
			}
		case xmlquery.ElementNode:
			childResult := nodeToJSONInternal(child, depth-1)
			addToResult(result, child.Data, childResult)
		}
	}

	if len(textParts) > 0 {
		if result.Len() == 0 {
			return strings.Join(textParts, "\n")
		}
		result.Set("#text", strings.Join(textParts, "\n"))
	}

	return result
}

func getTextContent(node *xmlquery.Node) string {
	var parts []string
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case xmlquery.TextNode, xmlquery.CharDataNode:
			text := strings.TrimSpace(child.Data)
			if text != "" {
				parts = append(parts, text)
			}
		case xmlquery.ElementNode:
			parts = append(parts, getTextContent(child))
		}
	}
	return strings.Join(parts, "\n")
}

func addToResult(result *OrderedMap, key string, value interface{}) {
	if key == "" {
		return
	}
	if existing, ok := result.Get(key); ok {
		switch existing := existing.(type) {
		case []interface{}:
			result.Set(key, append(existing, value))
		default:
			result.Set(key, []interface{}{existing, value})
		}
	} else {
		result.Set(key, value)
	}
}
