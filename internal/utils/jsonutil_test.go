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

func TestSelfClosingTagsHaveNullContent(t *testing.T) {
	// Self-closing XML elements should have null content in JSON
	xmlInput := `<root><self-closing/></root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)

	rootMap, ok := resultMap["root"].(map[string]interface{})
	assert.True(t, ok)

	assert.Nil(t, rootMap["self-closing"], "Self-closing tag should have null content")
}

func TestWhitespaceOnlyTagsAreSelfClosing(t *testing.T) {
	// Establishes that the XML formatter treats whitespace-only elements as self-closing
	xmlInput := `<root><spaces>  </spaces><newlines>
</newlines><empty></empty><immediate-closing></immediate-closing></root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)

	rootMap, ok := resultMap["root"].(map[string]interface{})
	assert.True(t, ok)

	assert.Nil(t, rootMap["spaces"], "Spaces should have null content")
	assert.Nil(t, rootMap["newlines"], "Newlines should have null content")
	assert.Nil(t, rootMap["empty"], "Empty should have null content")
	assert.Nil(t, rootMap["immediate-closing"], "Immediate-closing should have null content")
}
