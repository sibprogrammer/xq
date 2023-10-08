package main

import (
	_ "embed"
	"fmt"
	"github.com/sibprogrammer/xq/cmd"
	"strings"
)

var (
	commit = "000000"
	date   = ""
)

//go:embed version
var version string

func main() {
	fullVersion := strings.TrimSpace(version)
	if date != "" {
		fullVersion += fmt.Sprintf(" (%s, %s)", date, commit)
	}
	cmd.Version = fullVersion
	cmd.Execute()
}
