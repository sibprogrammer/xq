package utils

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/assert"
)

func TestXmlToJSON(t *testing.T) {
	tests := []struct {
		unformattedFile string
		expectedFile    string
		depth           int
	}{
		{"unformatted.xml", "formatted.json", -1},
		{"unformatted2.xml", "formatted2.json", -1},
		{"unformatted3.xml", "formatted3.json", -1},
		{"unformatted4.xml", "formatted4.json", 1},
	}

	for _, testCase := range tests {
		inputFileName := path.Join("..", "..", "test", "data", "xml2json", testCase.unformattedFile)
		unformattedXmlReader := getFileReader(inputFileName)

		outputFileName := path.Join("..", "..", "test", "data", "xml2json", testCase.expectedFile)
		data, jsonReadErr := os.ReadFile(outputFileName)
		assert.Nil(t, jsonReadErr)
		expectedJson := string(data)

		node, parseErr := xmlquery.Parse(unformattedXmlReader)
		assert.Nil(t, parseErr)
		result := NodeToJSON(node, testCase.depth)
		jsonData, jsonMarshalErr := json.Marshal(result)
		assert.Nil(t, jsonMarshalErr)

		output := new(strings.Builder)
		formatErr := FormatJson(bytes.NewReader(jsonData), output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedJson, output.String())
	}
}

func TestExhaustiveNodeTypeHandling(t *testing.T) {
	// Test that all xmlquery node types are handled without panicking
	// This verifies our exhaustive switch statements work correctly

	xmlInput := `<?xml version="1.0"?>
<!DOCTYPE root>
<!-- This is a comment -->
<root>
	<element>text content</element>
	<cdata><![CDATA[raw & unescaped < > content]]></cdata>
	<!-- another comment inside -->
	<?processing-instruction data?>
	<mixed>text<child>more</child>tail</mixed>
</root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	// Should not panic - this exercises all the node types
	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	// Verify the result is a map
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)

	// Verify root element exists
	root, ok := resultMap["root"]
	assert.True(t, ok)

	rootMap, ok := root.(map[string]interface{})
	assert.True(t, ok)

	// Verify CDATA is preserved as text
	cdataElem, ok := rootMap["cdata"]
	assert.True(t, ok)
	assert.Contains(t, cdataElem, "raw & unescaped")
}
