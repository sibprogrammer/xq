package main

import (
	"fmt"
	"github.com/sibprogrammer/xq/cmd"
)

var (
	commit  = "000000"
	date    = "unknown"
	version = "0.0.0"
)

func main() {
	cmd.Version = fmt.Sprintf("%s (%s, %s)", version, date, commit)
	cmd.Execute()
}
