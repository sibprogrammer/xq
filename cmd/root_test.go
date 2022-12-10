package cmd

import (
	"bytes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"path"
	"strings"
	"testing"
)

func execute(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()

	return strings.TrimSpace(buf.String()), err
}

func TestRootCmd(t *testing.T) {
	command := NewRootCmd()
	InitFlags(command)

	var output string
	var err error
	xmlFilePath := path.Join("..", "test", "data", "xml", "unformatted.xml")
	htmlFilePath := path.Join("..", "test", "data", "html", "unformatted.html")

	output, err = execute(command, xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "This is not a real user")

	output, err = execute(command, "-m", htmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "<html>")

	output, err = execute(command, "-q", "body > p", htmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "text")

	output, err = execute(command, "-x", "/user/@status", xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "active")
}
