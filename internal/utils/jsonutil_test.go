package utils

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/google/go-cmp/cmp"
)

func TestNodeToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		depth    int
		expected string
	}{
		{
			name:     "Simple XML",
			input:    "<root><child>value</child></root>",
			depth:    -1,
			expected: `{"root":{"child":"value"}}`,
		},
		{
			name:     "XML with attributes",
			input:    "<root attr=\"value\"><child>text</child></root>",
			depth:    -1,
			expected: `{"root":{"@attr":"value","child":"text"}}`,
		},
		{
			name:     "XML with mixed content",
			input:    "<root>\n  text  <child>value</child>\n  more text\n</root>",
			depth:    -1,
			expected: `{"root":{"#text":"text\nmore text","child":"value"}}`,
		},
		{
			name:     "Depth limited XML",
			input:    "<root><child1><grandchild>value</grandchild></child1><child2>text</child2></root>",
			depth:    2,
			expected: `{"root":{"child1":{"grandchild":"value"},"child2":"text"}}`,
		},
		{
			name:     "Depth 1 XML",
			input:    "<root><child1><grandchild>value</grandchild></child1><child2>text</child2></root>",
			depth:    1,
			expected: `{"root":{"child1":"value","child2":"text"}}`,
		},
		{
			name:     "Depth 0 XML",
			input:    "<root><child1><grandchild>value</grandchild></child1><child2>text</child2></root>",
			depth:    0,
			expected: `{"root":"value\ntext"}`,
		},
		{
			name: "mixed text and xml",
			input: `Thank you
<thinking>
1. woop
</thinking>

Bye`,
			expected: `{"#text":"Thank you\nBye","thinking":"1. woop"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := xmlquery.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			result := NodeToJSON(doc, tt.depth)
			resultJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal result to JSON: %v", err)
			}

			var resultMap, expectedMap map[string]interface{}
			err = json.Unmarshal(resultJSON, &resultMap)
			if err != nil {
				t.Fatalf("Failed to unmarshal result JSON: %v", err)
			}
			err = json.Unmarshal([]byte(tt.expected), &expectedMap)
			if err != nil {
				t.Fatalf("Failed to unmarshal expected JSON: %v", err)
			}

			t.Log(string(resultJSON))
			if diff := cmp.Diff(expectedMap, resultMap); diff != "" {
				t.Errorf("NodeToJSON mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
