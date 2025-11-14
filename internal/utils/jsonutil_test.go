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

func TestElementOrderInJSON(t *testing.T) {
	// Test that element order is preserved in JSON output
	xmlInput := `<root>
		<command-name>/status</command-name>
		<command-message>status</command-message>
		<command-args></command-args>
	</root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	// Marshal to JSON and check order
	jsonData, err := json.Marshal(result)
	assert.NoError(t, err)

	jsonStr := string(jsonData)
	t.Logf("JSON output: %s", jsonStr)

	// Find positions of each key in the JSON string
	namePos := strings.Index(jsonStr, "command-name")
	messagePos := strings.Index(jsonStr, "command-message")
	argsPos := strings.Index(jsonStr, "command-args")

	// Check that they appear in the original order
	assert.True(t, namePos < messagePos, "command-name should appear before command-message")
	assert.True(t, messagePos < argsPos, "command-message should appear before command-args")
}
