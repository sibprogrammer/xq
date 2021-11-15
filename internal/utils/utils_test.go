package utils

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path"
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
		unformattedXml := fileGetContents(path.Join("..", "..", "test", "data", unformattedFile))
		expectedXml := fileGetContents(path.Join("..", "..", "test", "data", expectedFile))

		formattedXml, err := FormatXml(unformattedXml)
		assert.Nil(t, err)
		assert.Equal(t, expectedXml, formattedXml)
	}
}

func fileGetContents(filename string) string {
	bytes, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	return string(bytes)
}
