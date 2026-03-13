package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func execute(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	if len(args) > 0 {
		cmd.SetArgs(args)
	} else {
		cmd.SetArgs([]string{})
	}

	err := cmd.Execute()

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
	})

	return strings.TrimSpace(buf.String()), err
}

func TestRootCmd(t *testing.T) {
	command := NewRootCmd()
	InitFlags(command)

	var output string
	var err error
	xmlFilePath := filepath.Join("..", "test", "data", "xml", "unformatted.xml")
	formattedXmlFilePath := filepath.Join("..", "test", "data", "xml", "formatted.xml")
	htmlFilePath := filepath.Join("..", "test", "data", "html", "unformatted.html")
	jsonFilePath := filepath.Join("..", "test", "data", "json", "unformatted.json")

	output, err = execute(command)
	assert.Nil(t, err)
	assert.Contains(t, output, "Usage:")

	output, err = execute(command, "--in-place", formattedXmlFilePath)
	assert.Nil(t, err)
	assert.Equal(t, output, "")

	output, err = execute(command, xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "This is not a real user")

	output, err = execute(command, "--no-color", xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "first_name")

	output, err = execute(command, "--indent", "0", xmlFilePath)
	assert.Nil(t, err)
	assert.NotContains(t, output, "\n")

	output, err = execute(command, jsonFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "{")

	output, err = execute(command, "--tab", xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "\t")

	output, err = execute(command, "-m", htmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "<html>")

	output, err = execute(command, "-q", "body > p", htmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "text")

	output, err = execute(command, "-x", "/user/@status", xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "active")

	output, err = execute(command, "--no-color", "-x", "/user/@status", xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "active")

	output, err = execute(command, "--color", "-x", "/user/@status", xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "active")

	_, err = execute(command, "nonexistent.xml")
	assert.ErrorContains(t, err, "no such file or directory")

	_, err = execute(command, "--indent", "-1", xmlFilePath)
	assert.ErrorContains(t, err, "indent should be")

	_, err = execute(command, "--indent", "incorrect", xmlFilePath)
	assert.ErrorContains(t, err, "invalid argument")
}

func TestCDATASupport(t *testing.T) {
	input := "<root><![CDATA[1 & 2]]></root>"
	doc, err := xmlquery.Parse(strings.NewReader(input))
	assert.Nil(t, err)

	result := utils.NodeToJSON(doc, 10)
	expected := map[string]interface{}{"root": "1 & 2"}

	assert.Equal(t, expected, result)
}

func TestEscapedTextNodes(t *testing.T) {
	// Test case 1: ampersand entity - reproduce issue #160
	// xq outputs bare & which fails when parsed as XML (used by -j flag)
	t.Run("ampersand entity output is valid XML", func(t *testing.T) {
		input := "<html>1 &amp; 2</html>"

		// First pass: format the HTML
		reader1 := strings.NewReader(input)
		var output1 bytes.Buffer
		err := utils.FormatHtml(reader1, &output1, "", utils.ColorsDisabled)
		assert.Nil(t, err)

		result1 := strings.TrimSpace(output1.String())
		t.Logf("First pass output: %q", result1)

		// Second pass: try to parse as XML (this is what `xq -j` does)
		reader2 := strings.NewReader(result1)
		_, err = xmlquery.Parse(reader2)
		assert.Nil(t, err, "xq output should be parseable as XML (for -j flag)")
	})

	// Test case 2: less-than and greater-than entities - reproduce issue #160
	// xq outputs bare < and > which are parsed as tags
	t.Run("less-than and greater-than entities output is valid XML", func(t *testing.T) {
		input := "<html>is &lt;bold&gt; a valid tag?</html>"

		// First pass: format the HTML
		reader1 := strings.NewReader(input)
		var output1 bytes.Buffer
		err := utils.FormatHtml(reader1, &output1, "", utils.ColorsDisabled)
		assert.Nil(t, err)

		result1 := strings.TrimSpace(output1.String())
		t.Logf("First pass output: %q", result1)

		// Second pass: try to parse as XML (this is what `xq -j` does)
		reader2 := strings.NewReader(result1)
		doc, err := xmlquery.Parse(reader2)
		assert.Nil(t, err, "xq output should be parseable as XML (for -j flag)")

		// Verify the text content is preserved correctly
		if doc != nil {
			textNode := xmlquery.FindOne(doc, "//html")
			if textNode != nil {
				assert.Equal(t, "is <bold> a valid tag?", textNode.InnerText(),
					"Text content should preserve the literal < and > characters")
			}
		}
	})
}

func TestProcessAsJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contentType utils.ContentType
		flags       map[string]interface{}
		expected    map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "Simple XML",
			input:       "<root><child>value</child></root>",
			contentType: utils.ContentXml,
			expected: map[string]interface{}{
				"root": map[string]interface{}{
					"child": "value",
				},
			},
		},
		{name: "Simple JSON",
			input:       `{"root": {"child": "value"}}`,
			contentType: utils.ContentJson,
			expected: map[string]interface{}{
				"root": map[string]interface{}{
					"child": "value",
				},
			},
		},
		{
			name:        "Simple HTML",
			input:       "<html><body><p>text</p></body></html>",
			contentType: utils.ContentHtml,
			expected: map[string]interface{}{
				"html": map[string]interface{}{
					"body": map[string]interface{}{
						"p": "text",
					},
				},
			},
		},
		{
			name:        "Plain text",
			input:       "text",
			contentType: utils.ContentText,
			expected: map[string]interface{}{
				"text": "text",
			},
		},
		{
			name:    "invalid input",
			input:   "thinking>\nI'll analyze each command and its output:\n</thinking>",
			wantErr: true,
		},
		{
			name: "combined",
			expected: map[string]interface{}{
				"#text":    "Thank you\nBye.",
				"thinking": "1. woop",
			},
			input: `Thank you
<thinking>
1. woop
</thinking>

Bye.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up flags
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.Bool("compact", false, "")
			flags.Int("depth", -1, "")
			for name, v := range tt.flags {
				_ = flags.Set(name, fmt.Sprint(v))
			}

			reader := strings.NewReader(tt.input)
			var output bytes.Buffer

			err := processAsJSON(flags, reader, &output, tt.contentType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				var resultMap map[string]interface{}
				err = json.Unmarshal(output.Bytes(), &resultMap)
				assert.NoError(t, err)

				assert.Equal(t, tt.expected, resultMap)
			}
		})
	}
}
