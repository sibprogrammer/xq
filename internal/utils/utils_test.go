package utils

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestFormatXml(t *testing.T) {
	unformattedXml := fileGetContents("../../test/data/unformatted.xml")
	expectedXml := fileGetContents("../../test/data/formatted.xml")

	formattedXml, err := FormatXml(unformattedXml)
	assert.Nil(t, err)
	assert.Equal(t, expectedXml, formattedXml)
}

func fileGetContents(filename string) string {
	bytes, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	return string(bytes)
}