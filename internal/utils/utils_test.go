package utils

import (
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
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
	}

	for unformattedFile, expectedFile := range files {
		unformattedXmlReader := getFileReader(path.Join("..", "..", "test", "data", unformattedFile))

		bytes, readErr := ioutil.ReadFile(path.Join("..", "..", "test", "data", expectedFile))
		assert.Nil(t, readErr)
		expectedXml := string(bytes)

		output := new(strings.Builder)
		formatErr := FormatXml(unformattedXmlReader, output, "  ")
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedXml, output.String())
	}
}

func getFileReader(filename string) io.Reader {
	reader, err := os.Open(filename)

	if err != nil {
		panic(err)
	}

	return reader
}
