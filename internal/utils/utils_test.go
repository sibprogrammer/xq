package utils

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path"
	"strings"
	"testing"
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
		"unformatted.xml":  "formatted.xml",
		"unformatted2.xml": "formatted2.xml",
		"unformatted3.xml": "formatted3.xml",
		"unformatted4.xml": "formatted4.xml",
		"unformatted5.xml": "formatted5.xml",
		"unformatted6.xml": "formatted6.xml",
		"unformatted7.xml": "formatted7.xml",
		"unformatted8.xml": "formatted8.xml",
		"unformatted9.xml": "formatted9.xml",
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
		"unformatted.xml":   "formatted.xml",
	}

	for unformattedFile, expectedFile := range files {
		unformattedHtmlReader := getFileReader(path.Join("..", "..", "test", "data", "html", unformattedFile))

		bytes, readErr := os.ReadFile(path.Join("..", "..", "test", "data", "html", expectedFile))
		assert.Nil(t, readErr)
		expectedHtml := string(bytes)

		output := new(strings.Builder)
		formatErr := FormatHtml(unformattedHtmlReader, output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedHtml, output.String())
	}
}

func TestXPathQuery(t *testing.T) {
	fileReader := getFileReader(path.Join("..", "..", "test", "data", "xml", "formatted.xml"))
	output := new(strings.Builder)
	err := XPathQuery(fileReader, output, "//first_name", true)
	assert.Nil(t, err)
	assert.Equal(t, "John", strings.Trim(output.String(), "\n"))
}

func TestCSSQuery(t *testing.T) {
	fileReader := getFileReader(path.Join("..", "..", "test", "data", "html", "formatted.html"))
	output := new(strings.Builder)
	err := CSSQuery(fileReader, output, "body > p")
	assert.Nil(t, err)
	assert.Equal(t, "text", strings.Trim(output.String(), "\n"))
}

func TestIsHTML(t *testing.T) {
	assert.True(t, IsHTML("<html>"))
	assert.True(t, IsHTML("<!doctype>"))
	assert.True(t, IsHTML("<body> ..."))

	assert.False(t, IsHTML("<?xml ?>"))
	assert.False(t, IsHTML("<root></root>"))
}

func TestPagerPrint(t *testing.T) {
	var output bytes.Buffer
	fileReader := getFileReader(path.Join("..", "..", "test", "data", "html", "formatted.html"))
	err := PagerPrint(fileReader, &output)
	assert.Nil(t, err)
	assert.Contains(t, output.String(), "<html>")
}
