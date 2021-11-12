package cmd

import (
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

// Version information
var Version string

var rootCmd = &cobra.Command{
	Use: "xq",
	Short: "Command line XML beautifier and content extractor",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var bytes []byte
		var err error
		var result string
		query, _ := cmd.Flags().GetString("xpath")


		if len(args) == 0 {
			fileInfo, _ := os.Stdin.Stat()

			if (fileInfo.Mode() & os.ModeCharDevice) != 0 {
				_ = cmd.Help()
				return nil
			}

			bytes, err = ioutil.ReadAll(os.Stdin)
		} else {
			bytes, err = ioutil.ReadFile(args[len(args)-1])
		}

		if err != nil {
			return err
		}

		if query != "" {
			result, err = utils.XPathQuery(string(bytes), query)
		} else {
			result, err = utils.FormatXml(string(bytes))
		}

		if err != nil {
			return err
		}

		utils.PagerPrint(result)
		return nil
	},
}

func Execute() {
	rootCmd.Version = Version
	rootCmd.PersistentFlags().StringP("xpath", "x", "", "Extract the node(s) from XML")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}