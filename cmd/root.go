package cmd

import (
	"fmt"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "xq",
	Short: "An XML prettier and content extractor",
	Run: func(cmd *cobra.Command, args []string) {
		var bytes []byte
		var err error

		if len(args) == 0 {
			fileInfo, _ := os.Stdin.Stat()

			if (fileInfo.Mode() & os.ModeCharDevice) != 0 {
				_ = cmd.Help()
				return
			}

			bytes, err = ioutil.ReadAll(os.Stdin)
		} else {
			bytes, err = ioutil.ReadFile(args[len(args)-1])
		}

		if err != nil {
			log.Fatal("Unable to read the input from stdin:", err)
		}

		fmt.Println(utils.FormatXml(string(bytes)))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}