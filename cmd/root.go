package cmd

import (
	"errors"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io/ioutil"
	"os"
	"strings"
)

// Version information
var Version string

var rootCmd = &cobra.Command{
	Use:          "xq",
	Short:        "Command line XML beautifier and content extractor",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var bytes []byte
		var err error
		var result string
		query, _ := cmd.Flags().GetString("xpath")

		indent, err := getIndent(cmd.Flags())
		if err != nil {
			return err
		}

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
			result, err = utils.FormatXml(string(bytes), indent)
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
	rootCmd.PersistentFlags().Bool("tab", false, "Use tabs for indentation")
	rootCmd.PersistentFlags().Int("indent", 2, "Use the given number of spaces for indentation")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getIndent(flags *pflag.FlagSet) (string, error) {
	var indentWidth int
	var tabIndent bool
	var err error

	indentWidth, err = flags.GetInt("indent")
	if err != nil {
		return "", err
	}
	if indentWidth < 1 || indentWidth > 8 {
		return "", errors.New("intend should be between 1-8 spaces")
	}

	indent := strings.Repeat(" ", indentWidth)

	tabIndent, err = flags.GetBool("tab")
	if err != nil {
		return "", err
	}

	if tabIndent {
		indent = "\t"
	}

	return indent, nil
}
