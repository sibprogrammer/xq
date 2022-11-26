package utils

import (
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path"
	"strings"
	"testing"
)

func TestFormatXml(t *testing.T) {
	files := map[string]string{
		"unformatted.xml":  "formatted.xml",
		"unformatted2.xml": "formatted2.xml",
		"unformatted3.xml": "formatted3.xml",
		"unformatted4.xml": "formatted4.xml",
		"unformatted5.xml": "formatted5.xml",
		"unformatted6.xml": "formatted6.xml",
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

func getFileReader(filename string) io.Reader {
	reader, err := os.Open(filename)

	if err != nil {
		panic(err)
	}

	return reader
}
