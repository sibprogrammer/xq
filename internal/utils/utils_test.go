package utils

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	}

	for unformattedFile, expectedFile := range files {
		unformattedXmlReader := getFileReader(path.Join("..", "..", "test", "data", "xml", unformattedFile))

		bytes, readErr := os.ReadFile(path.Join("..", "..", "test", "data", "xml", expectedFile))
		assert.Nil(t, readErr)
		expectedXml := string(bytes)

		output := new(strings.Builder)
		formatErr := FormatXml(unformattedXmlReader, output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedXml, output.String())
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
		"unformatted.xml":   "formatted.xml",
	}

	for unformattedFile, expectedFile := range files {
		unformattedHtmlReader := getFileReader(path.Join("..", "..", "test", "data", "html", unformattedFile))

		data, readErr := os.ReadFile(path.Join("..", "..", "test", "data", "html", expectedFile))
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
		unformattedJsonReader := getFileReader(path.Join("..", "..", "test", "data", "json", unformattedFile))

		data, readErr := os.ReadFile(path.Join("..", "..", "test", "data", "json", expectedFile))
		assert.Nil(t, readErr)
		expectedJson := string(data)

		output := new(strings.Builder)
		formatErr := FormatJson(unformattedJsonReader, output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedJson, output.String())
	}
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
		{input: "unformatted8.xml", node: false, single: true, query: "//title", result: "Some Title"},
		{input: "unformatted8.xml", node: true, single: true, query: "//title", result: "<title>Some Title</title>"},
		{input: "unformatted8.xml", node: false, single: false, query: "count(//link)", result: "2"},
	}

	for _, testCase := range tests {
		fileReader := getFileReader(path.Join("..", "..", "test", "data", "xml", testCase.input))
		output := new(strings.Builder)
		options := QueryOptions{WithTags: testCase.node, Indent: "  "}
		err := XPathQuery(fileReader, output, testCase.query, testCase.single, options)
		assert.Nil(t, err)
		assert.Equal(t, testCase.result, strings.Trim(output.String(), "\n"))
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
		fileReader := getFileReader(path.Join("..", "..", "test", "data", "html", testCase.input))
		output := new(strings.Builder)
		options := QueryOptions{WithTags: testCase.node, Indent: "  "}
		err := CSSQuery(fileReader, output, testCase.query, testCase.attr, options)
		assert.Nil(t, err)
		assert.Equal(t, testCase.result, strings.Trim(output.String(), "\n"))
	}
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
}

func TestPagerPrint(t *testing.T) {
	var output bytes.Buffer
	fileReader := getFileReader(path.Join("..", "..", "test", "data", "html", "formatted.html"))
	err := PagerPrint(fileReader, &output)
	assert.Nil(t, err)
	assert.Contains(t, output.String(), "<html>")
}

func TestEscapeText(t *testing.T) {
	result, err := escapeText("\"value\"")
	assert.Nil(t, err)
	assert.Equal(t, "&quot;value&quot;", result)
}
