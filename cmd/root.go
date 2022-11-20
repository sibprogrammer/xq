package cmd

import (
	"errors"
	"fmt"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io"
	"os"
	"strings"
)

// Version information
var Version string

var rootCmd = &cobra.Command{
	Use:          "xq",
	Short:        "Command-line XML and HTML beautifier and content extractor",
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

		xPathQuery, singleNode := getXpathQuery(cmd.Flags())
		cssQuery, _ := cmd.Flags().GetString("query")

		pr, pw := io.Pipe()

		go func() {
			defer pw.Close()

			if xPathQuery != "" {
				err = utils.XPathQuery(reader, pw, xPathQuery, singleNode)
			} else if cssQuery != "" {
				err = utils.CSSQuery(reader, pw, cssQuery)
			} else {
				colors := getColorMode(cmd.Flags())
				isHtmlFormatter, _ := cmd.Flags().GetBool("html")
				if isHtmlFormatter {
					err = utils.FormatHtml(reader, pw, indent, colors)
				} else {
					err = utils.FormatXml(reader, pw, indent, colors)
				}
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
	if err := initViper(); err != nil {
		fmt.Printf("Error while reading the config file: %v\n", err)
		os.Exit(1)
	}

	rootCmd.Version = Version

	rootCmd.Flags().BoolP("help", "h", false, "Print this help message")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
	rootCmd.PersistentFlags().StringP("xpath", "x", "", "Extract the node(s) from XML")
	rootCmd.PersistentFlags().StringP("extract", "e", "", "Extract a single node from XML")
	rootCmd.PersistentFlags().Bool("tab", viper.GetBool("tab"), "Use tabs for indentation")
	rootCmd.PersistentFlags().Int("indent", viper.GetInt("indent"),
		"Use the given number of spaces for indentation")
	rootCmd.PersistentFlags().Bool("no-color", viper.GetBool("no-color"), "Disable colorful output")
	rootCmd.PersistentFlags().BoolP("color", "c", viper.GetBool("color"),
		"Force colorful output")
	rootCmd.PersistentFlags().BoolP("html", "m", viper.GetBool("html"), "Use HTML formatter")
	rootCmd.PersistentFlags().StringP("query", "q", "",
		"Extract the node(s) using CSS selectors")

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

func initViper() error {
	viper.SetConfigName(".xq")
	viper.SetConfigType("env")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	viper.SetDefault("indent", 2)
	viper.SetDefault("tab", false)
	viper.SetDefault("no-color", false)
	viper.SetDefault("color", false)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

func getXpathQuery(flags *pflag.FlagSet) (query string, single bool) {
	if query, _ = flags.GetString("xpath"); query != "" {
		return query, false
	}

	query, _ = flags.GetString("extract")
	return query, true
}

func getColorMode(flags *pflag.FlagSet) int {
	colors := utils.ColorsDefault

	disableColors, _ := flags.GetBool("no-color")
	if disableColors {
		colors = utils.ColorsDisabled
	}

	forcedColors, _ := flags.GetBool("color")
	if forcedColors {
		colors = utils.ColorsForced
	}

	return colors
}
