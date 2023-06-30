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
	if len(args) > 0 {
		cmd.SetArgs(args)
	} else {
		cmd.SetArgs([]string{})
	}

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

	output, err = execute(command)
	assert.Contains(t, output, "Usage:")

	output, err = execute(command, xmlFilePath)
	assert.Nil(t, err)
	assert.Contains(t, output, "This is not a real user")

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

	_, err = execute(command, "--indent", "0", xmlFilePath)
	assert.ErrorContains(t, err, "indent should be")

	_, err = execute(command, "--indent", "incorrect", xmlFilePath)
	assert.ErrorContains(t, err, "invalid argument")
}
