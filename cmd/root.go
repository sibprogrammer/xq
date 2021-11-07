package cmd

import (
	"fmt"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
)

// Version information
var Version string

var rootCmd = &cobra.Command{
	Use: "xq",
	Short: "An XML prettier and content extractor",
	Run: func(cmd *cobra.Command, args []string) {
		var bytes []byte
		var err error
		query, _ := cmd.Flags().GetString("xpath")


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
			log.Fatal("Unable to read the input:", err)
		}

		if query != "" {
			fmt.Print(utils.XPathQuery(string(bytes), query))
		} else {
			fmt.Println(utils.FormatXml(string(bytes)))
		}
	},
}

func Execute() {
	rootCmd.Version = Version
	rootCmd.PersistentFlags().StringP("xpath", "x", "", "Extract the node(s) from XML")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}