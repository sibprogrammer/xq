package cmd

import (
	"errors"
	"fmt"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
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
		var err error
		var reader io.Reader
		var indent string

		if indent, err = getIndent(cmd.Flags()); err != nil {
			return err
		}

		if len(args) == 0 {
			fileInfo, _ := os.Stdin.Stat()

			if (fileInfo.Mode() & os.ModeCharDevice) != 0 {
				_ = cmd.Help()
				return nil
			}

			reader = os.Stdin
		} else {
			if reader, err = os.Open(args[len(args)-1]); err != nil {
				return err
			}
		}

		query, _ := cmd.Flags().GetString("xpath")
		pr, pw := io.Pipe()

		go func() {
			defer pw.Close()

			if query != "" {
				err = utils.XPathQuery(reader, pw, query)
			} else {
				err = utils.FormatXml(reader, pw, indent)
			}

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}()

		return utils.PagerPrint(pr)
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

	if indentWidth, err = flags.GetInt("indent"); err != nil {
		return "", err
	}
	if indentWidth < 1 || indentWidth > 8 {
		return "", errors.New("intend should be between 1-8 spaces")
	}

	indent := strings.Repeat(" ", indentWidth)

	if tabIndent, err = flags.GetBool("tab"); err != nil {
		return "", err
	}

	if tabIndent {
		indent = "\t"
	}

	return indent, nil
}
