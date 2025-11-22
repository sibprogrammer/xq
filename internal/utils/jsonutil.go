package utils

import (
	"strings"

	"github.com/antchfx/xmlquery"
)

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
		result := make(map[string]interface{})
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
				result[child.Data] = childResult
			case xmlquery.TextNode, xmlquery.CharDataNode:
				text := strings.TrimSpace(child.Data)
				if text != "" {
					textParts = append(textParts, text)
				}
			}
		}

		if len(textParts) > 0 {
			result["#text"] = strings.Join(textParts, "\n")
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

	result := make(map[string]interface{})
	for _, attr := range node.Attr {
		result["@"+attr.Name.Local] = attr.Value
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
		if len(result) == 0 {
			// Element contains only text
			return strings.Join(textParts, "\n")
		}
		result["#text"] = strings.Join(textParts, "\n")
	} else if len(result) == 0 {
		// Self-closing tags have null content
		return nil
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

func addToResult(result map[string]interface{}, key string, value interface{}) {
	if key == "" {
		return
	}
	if existing, ok := result[key]; ok {
		switch existing := existing.(type) {
		case []interface{}:
			result[key] = append(existing, value)
		default:
			result[key] = []interface{}{existing, value}
		}
	} else {
		result[key] = value
	}
}
