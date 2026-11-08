package utils

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

type failOnWriteWriter struct {
	count  int
	failOn int
}

func (writer *failOnWriteWriter) Write(data []byte) (int, error) {
	writer.count++
	if writer.count == writer.failOn {
		return 0, errWriteFailed
	}

	return len(data), nil
}

func getFileReader(filename string) io.Reader {
	reader, err := os.Open(filename)

	if err != nil {
		panic(err)
	}

	return reader
}

func TestFormatXml(t *testing.T) {
	files := map[string]string{
		"unformatted.xml":   "formatted.xml",
		"unformatted2.xml":  "formatted2.xml",
		"unformatted3.xml":  "formatted3.xml",
		"unformatted4.xml":  "formatted4.xml",
		"unformatted5.xml":  "formatted5.xml",
		"unformatted6.xml":  "formatted6.xml",
		"unformatted7.xml":  "formatted7.xml",
		"unformatted8.xml":  "formatted8.xml",
		"unformatted9.xml":  "formatted9.xml",
		"unformatted10.xml": "formatted10.xml",
		"unformatted11.xml": "formatted11.xml",
		"unformatted12.xml": "formatted12.xml",
		"unformatted13.xml": "formatted13.xml",
		"unformatted14.xml": "formatted14.xml",
		"unformatted15.xml": "formatted15.xml",
		"unformatted16.xml": "formatted16.xml",
		"unformatted17.xml": "formatted17.xml",
		"unformatted18.xml": "formatted18.xml",
		"unformatted19.xml": "formatted19.xml",
		"unformatted20.xml": "formatted20.xml",
	}

	for unformattedFile, expectedFile := range files {
		unformattedXmlReader := getFileReader(filepath.Join("..", "..", "test", "data", "xml", unformattedFile))

		bytes, readErr := os.ReadFile(filepath.Join("..", "..", "test", "data", "xml", expectedFile))
		assert.Nil(t, readErr)
		expectedXml := string(bytes)

		output := new(strings.Builder)
		formatErr := FormatXml(unformattedXmlReader, output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedXml, output.String())
	}
}

func TestFormatXmlCDATA(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"issue example", "<tag><![CDATA[some text>]]></tag>", "<tag><![CDATA[some text>]]></tag>"},
		{"plain text", "<tag><![CDATA[some text]]></tag>", "<tag><![CDATA[some text]]></tag>"},
		{"special characters", "<tag><![CDATA[<p>&amp;</p>]]></tag>", "<tag><![CDATA[<p>&amp;</p>]]></tag>"},
		{"empty", "<tag><![CDATA[]]></tag>", "<tag><![CDATA[]]></tag>"},
		{"whitespace", "<tag><![CDATA[ \t\n  ]]></tag>", "<tag><![CDATA[ \t\n  ]]></tag>"},
		{"line endings", "<tag><![CDATA[\r\n  text \r\n]]></tag>", "<tag><![CDATA[\r\n  text \r\n]]></tag>"},
		{"adjacent sections", "<tag><![CDATA[one]]><![CDATA[]]><![CDATA[two]]></tag>", "<tag><![CDATA[one]]><![CDATA[]]><![CDATA[two]]></tag>"},
		{"mixed text", "<tag>before<![CDATA[ inside ]]>after</tag>", "<tag>before<![CDATA[ inside ]]>after</tag>"},
		{"entities", "<tag>&#x41;<![CDATA[text]]>&#x42;</tag>", "<tag>A<![CDATA[text]]>B</tag>"},
		{"trailing whitespace", "<tag><![CDATA[text]]>\n  </tag>", "<tag><![CDATA[text]]></tag>"},
		{"mixed elements", "<root><tag/><![CDATA[ text ]]><tag/></root>", "<root>\n  <tag/><![CDATA[ text ]]>\n  <tag/>\n</root>"},
		{"XML 1.1", "<?xml version='1.1'?><tag><![CDATA[text]]></tag>", "<?xml version='1.1'?>\n<tag><![CDATA[text]]></tag>"},
		{"converted charset", "<?xml version='1.0' encoding='windows-1251'?><tag>\xcf<![CDATA[\xf0\xe8\xe2\xe5\xf2]]></tag>", "<?xml version='1.0' encoding='windows-1251'?>\n<tag>П<![CDATA[ривет]]></tag>"},
		{"long token", "<tag>" + strings.Repeat("x", 8192) + "<![CDATA[text]]></tag>", "<tag>" + strings.Repeat("x", 8192) + "<![CDATA[text]]></tag>"},
		{"long converted token", "<?xml version='1.0' encoding='windows-1251'?><tag>" + strings.Repeat("\xcf", 8192) + "<![CDATA[\xf0\xe8\xe2\xe5\xf2]]></tag>", "<?xml version='1.0' encoding='windows-1251'?>\n<tag>" + strings.Repeat("П", 8192) + "<![CDATA[ривет]]></tag>"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			for _, chunked := range []bool{false, true} {
				var reader io.Reader = strings.NewReader(testCase.input)
				if chunked {
					reader = iotest.OneByteReader(reader)
				}
				output := new(strings.Builder)
				err := FormatXml(reader, output, "  ", ColorsDisabled)
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected+"\n", output.String())
			}
		})
	}
}

func TestFormatHtml(t *testing.T) {
	files := map[string]string{
		"unformatted.html":  "formatted.html",
		"unformatted2.html": "formatted2.html",
		"unformatted3.html": "formatted3.html",
		"unformatted4.html": "formatted4.html",
		"unformatted5.html": "formatted5.html",
		"unformatted6.html": "formatted6.html",
		"unformatted7.html": "formatted7.html",
		"unformatted.xml":   "formatted.xml",
	}

	for unformattedFile, expectedFile := range files {
		unformattedHtmlReader := getFileReader(filepath.Join("..", "..", "test", "data", "html", unformattedFile))

		data, readErr := os.ReadFile(filepath.Join("..", "..", "test", "data", "html", expectedFile))
		assert.Nil(t, readErr)
		expectedHtml := string(data)

		output := new(strings.Builder)
		formatErr := FormatHtml(unformattedHtmlReader, output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedHtml, output.String())
	}
}

func TestFormatJson(t *testing.T) {
	files := map[string]string{
		"unformatted.json":  "formatted.json",
		"unformatted2.json": "formatted2.json",
		"unformatted3.json": "formatted3.json",
	}

	for unformattedFile, expectedFile := range files {
		unformattedJsonReader := getFileReader(filepath.Join("..", "..", "test", "data", "json", unformattedFile))

		data, readErr := os.ReadFile(filepath.Join("..", "..", "test", "data", "json", expectedFile))
		assert.Nil(t, readErr)
		expectedJson := string(data)

		output := new(strings.Builder)
		formatErr := FormatJson(unformattedJsonReader, output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedJson, output.String())
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty object", "{}", "{}\n"},
		{"empty array", "[]", "[]\n"},
		{"nested empty containers in object", "{\"a\":{},\"b\":[]}", "{\n  \"a\": {},\n  \"b\": []\n}\n"},
		{"nested empty containers in array", "[{},[]]", "[\n  {},\n  []\n]\n"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			output := new(strings.Builder)
			formatErr := FormatJson(strings.NewReader(testCase.input), output, "  ", ColorsDisabled)
			assert.NoError(t, formatErr)
			assert.Equal(t, testCase.expected, output.String())
		})
	}

	formatErr := FormatJson(strings.NewReader("{}"), failingWriter{}, "  ", ColorsDisabled)
	assert.ErrorIs(t, formatErr, errWriteFailed)

	formatErr = FormatJson(strings.NewReader("{}"), &failOnWriteWriter{failOn: 3}, "  ", ColorsDisabled)
	assert.ErrorIs(t, formatErr, errWriteFailed)
}

func TestXPathQuery(t *testing.T) {
	type test struct {
		input  string
		node   bool
		single bool
		query  string
		result string
	}

	tests := []test{
		{input: "formatted.xml", node: false, single: true, query: "//first_name", result: "John"},
		{input: "unformatted19.xml", node: false, single: true, query: "//id", result: "18"},
		{input: "unformatted8.xml", node: false, single: true, query: "//title", result: "Some Title"},
		{input: "unformatted8.xml", node: true, single: true, query: "//title", result: "<title>Some Title</title>"},
		{input: "unformatted8.xml", node: false, single: false, query: "count(//link)", result: "2"},
	}

	for _, testCase := range tests {
		fileReader := getFileReader(filepath.Join("..", "..", "test", "data", "xml", testCase.input))
		output := new(strings.Builder)
		options := QueryOptions{WithTags: testCase.node, Indent: "  "}
		err := XPathQuery(fileReader, output, testCase.query, testCase.single, options)
		assert.Nil(t, err)
		assert.Equal(t, testCase.result, strings.Trim(output.String(), "\n"))
	}
}

func TestXPathQueryNoMatch(t *testing.T) {
	type test struct {
		node   bool
		single bool
		query  string
	}

	tests := []test{
		{node: false, single: true, query: "//missing"},
		{node: true, single: false, query: "//missing"},
		{node: false, single: false, query: "//missing"},
		{node: false, single: false, query: "string(//missing)"},
	}

	for _, testCase := range tests {
		output := new(strings.Builder)
		options := QueryOptions{WithTags: testCase.node, Indent: "  "}
		reader := strings.NewReader(`<?xml version="1.0"?><root></root>`)
		err := XPathQuery(reader, output, testCase.query, testCase.single, options)
		assert.ErrorIs(t, err, ErrNoMatch)
	}
}

func TestXPathQueryBoolean(t *testing.T) {
	tests := map[string]string{
		"boolean(//root)":    "true",
		"boolean(//missing)": "false",
	}

	for query, expected := range tests {
		output := new(strings.Builder)
		options := QueryOptions{}
		err := XPathQuery(strings.NewReader(`<?xml version="1.0"?><root></root>`), output, query, false, options)
		if expected == "true" {
			assert.Nil(t, err)
		} else {
			assert.ErrorIs(t, err, ErrNoMatch)
		}
		assert.Equal(t, expected, strings.Trim(output.String(), "\n"))
	}
}

func TestCSSQuery(t *testing.T) {
	type test struct {
		input  string
		node   bool
		query  string
		attr   string
		result string
	}

	tests := []test{
		{input: "formatted.html", node: false, query: "body > p", attr: "", result: "text"},
		{input: "formatted.html", node: false, query: "script", attr: "src", result: "foo.js\nbar.js\nbaz.js"},
		{input: "formatted.html", node: true, query: "p", attr: "", result: "<p>text</p>"},
		{input: "formatted.html", node: true, query: "a", attr: "", result: "<a href=\"https://example.com\">link</a>"},
	}

	for _, testCase := range tests {
		fileReader := getFileReader(filepath.Join("..", "..", "test", "data", "html", testCase.input))
		output := new(strings.Builder)
		options := QueryOptions{WithTags: testCase.node, Indent: "  "}
		err := CSSQuery(fileReader, output, testCase.query, testCase.attr, options)
		assert.Nil(t, err)
		assert.Equal(t, testCase.result, strings.Trim(output.String(), "\n"))
	}

	output := new(strings.Builder)
	err := CSSQuery(strings.NewReader("<html><body></body></html>"), output, "p", "", QueryOptions{})
	assert.ErrorIs(t, err, ErrNoMatch)
}

func TestIsHTML(t *testing.T) {
	assert.True(t, IsHTML("<html>"))
	assert.True(t, IsHTML("<!doctype>"))
	assert.True(t, IsHTML("<body> ..."))

	assert.False(t, IsHTML("<?xml ?>"))
	assert.False(t, IsHTML("<root></root>"))
}

func TestIsJSON(t *testing.T) {
	assert.True(t, IsJSON(`{"key": "value"}`))
	assert.True(t, IsJSON(`{"key": "value", "key2": "value2"}`))
	assert.True(t, IsJSON(`[1, 2, 3]`))
	assert.True(t, IsJSON(`   {}`))
	assert.False(t, IsJSON(`<html></html>`))
	assert.False(t, IsJSON(`<![CDATA[text]]>`))
}

func TestPagerPrint(t *testing.T) {
	var output bytes.Buffer
	fileReader := getFileReader(filepath.Join("..", "..", "test", "data", "html", "formatted.html"))
	err := PagerPrint(fileReader, &output, "")
	assert.Nil(t, err)
	assert.Contains(t, output.String(), "<html>")

	fileReader = getFileReader(filepath.Join("..", "..", "test", "data", "html", "formatted.html"))
	err = PagerPrint(fileReader, &output, "cat")
	assert.Nil(t, err)
	assert.Contains(t, output.String(), "<html>")
}

func TestSanitizeXmlVersion(t *testing.T) {
	tests := []struct {
		input           string
		expected        string
		expectedVersion string
	}{
		{`<?xml version="1.1"?><root/>`, `<?xml version="1.0"?><root/>`, "1.1"},
		{"<?xml version='1.1' encoding='UTF-8'?><root/>", "<?xml version='1.0' encoding='UTF-8'?><root/>", "1.1"},
		{"\ufeff<?xml version=\"1.1\"?><root/>", "\ufeff<?xml version=\"1.0\"?><root/>", "1.1"},
		{"garbage line\n<?xml version=\"1.1\"?><root/>", "garbage line\n<?xml version=\"1.0\"?><root/>", "1.1"},
		{`<root attr="1.1"/>`, `<root attr="1.1"/>`, ""},
		{`<?xml version="1.0"?><root/>`, `<?xml version="1.0"?><root/>`, ""},
		{`<root/>`, `<root/>`, ""},
		{``, ``, ""},
	}

	for _, testCase := range tests {
		reader, version := SanitizeXmlVersion(strings.NewReader(testCase.input))
		result, err := io.ReadAll(reader)
		assert.Nil(t, err)
		assert.Equal(t, testCase.expected, string(result))
		assert.Equal(t, testCase.expectedVersion, version)
	}
}

func TestEscapeText(t *testing.T) {
	result, err := escapeText("\"value\"")
	assert.Nil(t, err)
	assert.Equal(t, "&quot;value&quot;", result)
}
