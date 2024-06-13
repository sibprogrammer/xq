package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Version information
var Version string

var rootCmd = NewRootCmd()

type Config struct {
	indent  int
	tab     bool
	noColor bool
	color   bool
	html    bool
	node    bool
}

var config Config

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
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
			withTags, _ := cmd.Flags().GetBool("node")
			colors := getColorMode(cmd.Flags())

			options := utils.QueryOptions{
				WithTags: withTags,
				Indent:   indent,
				Colors:   colors,
			}

			cssQuery, _ := cmd.Flags().GetString("query")
			cssAttr, _ := cmd.Flags().GetString("attr")
			if cssAttr != "" && cssQuery == "" {
				return errors.New("query option (-q) is missed for attribute selection")
			}

			pr, pw := io.Pipe()

			go func() {
				defer pw.Close()

				if xPathQuery != "" {
					err = utils.XPathQuery(reader, pw, xPathQuery, singleNode, options)
				} else if cssQuery != "" {
					err = utils.CSSQuery(reader, pw, cssQuery, cssAttr, options)
				} else {
					var contentType utils.ContentType
					contentType, reader = detectFormat(cmd.Flags(), reader)

					switch contentType {
					case utils.ContentHtml:
						err = utils.FormatHtml(reader, pw, indent, colors)
					case utils.ContentXml:
						err = utils.FormatXml(reader, pw, indent, colors)
					case utils.ContentJson:
						err = utils.FormatJson(reader, pw, indent, colors)
					default:
						err = fmt.Errorf("unknown content type: %v", contentType)
					}
				}

				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}()

			return utils.PagerPrint(pr, cmd.OutOrStdout())
		},
	}
}

func InitFlags(cmd *cobra.Command) {
	if err := initConfig(); err != nil {
		fmt.Printf("Error while reading the config file: %v\n", err)
		os.Exit(1)
	}

	cmd.Version = Version

	cmd.Flags().BoolP("help", "h", false, "Print this help message")
	cmd.Flags().BoolP("version", "v", false, "Print version information")
	cmd.PersistentFlags().StringP("xpath", "x", "", "Extract the node(s) from XML")
	cmd.PersistentFlags().StringP("extract", "e", "", "Extract a single node from XML")
	cmd.PersistentFlags().Bool("tab", config.tab, "Use tabs for indentation")
	cmd.PersistentFlags().Int("indent", config.indent,
		"Use the given number of spaces for indentation")
	cmd.PersistentFlags().Bool("no-color", config.noColor, "Disable colorful output")
	cmd.PersistentFlags().BoolP("color", "c", config.color,
		"Force colorful output")
	cmd.PersistentFlags().BoolP("html", "m", config.html, "Use HTML formatter")
	cmd.PersistentFlags().StringP("query", "q", "",
		"Extract the node(s) using CSS selector")
	cmd.PersistentFlags().StringP("attr", "a", "",
		"Extract an attribute value instead of node content for provided CSS query")
	cmd.PersistentFlags().BoolP("node", "n", config.node,
		"Return the node content instead of text")
}

func Execute() {
	InitFlags(rootCmd)

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
	if indentWidth < 0 || indentWidth > 8 {
		return "", errors.New("indent should be between 0-8 spaces")
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

func initConfig() error {
	config.indent = 2
	config.tab = false
	config.noColor = false
	config.color = false
	config.html = false
	config.node = false

	homeDir, _ := os.UserHomeDir()

	file, err := os.Open(filepath.Join(homeDir, ".xq"))
	if os.IsNotExist(err) {
		return nil
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var text = scanner.Text()
		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "#") || len(text) == 0 {
			continue
		}
		var parts = strings.Split(text, "=")
		if len(parts) != 2 {
			continue
		}
		option, value := parts[0], parts[1]
		option = strings.TrimSpace(option)
		value = strings.TrimSpace(value)

		switch option {
		case "indent":
			config.indent, _ = strconv.Atoi(value)
		case "tab":
			config.tab, _ = strconv.ParseBool(value)
		case "no-color":
			config.noColor, _ = strconv.ParseBool(value)
		case "color":
			config.color, _ = strconv.ParseBool(value)
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

func detectFormat(flags *pflag.FlagSet, origReader io.Reader) (utils.ContentType, io.Reader) {
	isHtmlFormatter, _ := flags.GetBool("html")
	if isHtmlFormatter {
		return utils.ContentHtml, origReader
	}

	buf := make([]byte, 10)
	length, err := origReader.Read(buf)
	if err != nil {
		return utils.ContentText, origReader
	}

	reader := io.MultiReader(bytes.NewReader(buf[:length]), origReader)

	if utils.IsJSON(string(buf)) {
		return utils.ContentJson, reader
	}

	if utils.IsHTML(string(buf)) {
		return utils.ContentHtml, reader
	}

	return utils.ContentXml, reader
}
