package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/sibprogrammer/xq/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Version information
var Version string

var rootCmd = NewRootCmd()

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "xq",
		Short:        "Command-line XML and HTML beautifier and content extractor",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var reader io.Reader
			var indent string
			var path string

			inPlace, _ := cmd.Flags().GetBool("in-place")

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
				path = args[len(args)-1]
				var err error
				if reader, err = os.Open(path); err != nil {
					return err
				}
			}

			if path == "" && inPlace {
				return errors.New("in-place formatting requires a file path")
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
			jsonOutputMode, _ := cmd.Flags().GetBool("json")

			pr, pw := io.Pipe()
			errChan := make(chan error, 1)

			go func() {
				defer close(errChan)
				defer pw.Close()

				var err error
				if xPathQuery != "" {
					err = utils.XPathQuery(reader, pw, xPathQuery, singleNode, options)
				} else if cssQuery != "" {
					err = utils.CSSQuery(reader, pw, cssQuery, cssAttr, options)
				} else {
					var contentType utils.ContentType
					contentType, reader = detectFormat(cmd.Flags(), reader)
					if jsonOutputMode {
						err = processAsJSON(cmd.Flags(), reader, pw, contentType)
					} else {
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
				}

				errChan <- err
			}()

			if inPlace {
				var content []byte
				if content, err = io.ReadAll(pr); err != nil {
					return err
				}
				if err = os.WriteFile(path, content, 0600); err != nil {
					return err
				}
			} else {
				if err := utils.PagerPrint(pr, cmd.OutOrStdout()); err != nil {
					return err
				}
			}

			return <-errChan
		},
	}
}

func InitFlags(cmd *cobra.Command) {
	homeDir, _ := os.UserHomeDir()
	configFile := filepath.Join(homeDir, ".xq")
	if err := utils.LoadConfig(configFile); err != nil {
		fmt.Printf("Error while reading the config file: %v\n", err)
		os.Exit(1)
	}

	cmd.Version = Version

	cmd.Flags().BoolP("help", "h", false, "Print this help message")
	cmd.Flags().BoolP("version", "v", false, "Print version information")
	cmd.PersistentFlags().StringP("xpath", "x", "", "Extract the node(s) from XML")
	cmd.PersistentFlags().StringP("extract", "e", "", "Extract a single node from XML")
	cmd.PersistentFlags().Bool("tab", utils.GetConfig().Tab, "Use tabs for indentation")
	cmd.PersistentFlags().Int("indent", utils.GetConfig().Indent,
		"Use the given number of spaces for indentation")
	cmd.PersistentFlags().Bool("no-color", utils.GetConfig().NoColor, "Disable colorful output")
	cmd.PersistentFlags().BoolP("color", "c", utils.GetConfig().Color,
		"Force colorful output")
	cmd.PersistentFlags().BoolP("html", "m", utils.GetConfig().Html, "Use HTML formatter")
	cmd.PersistentFlags().StringP("query", "q", "",
		"Extract the node(s) using CSS selector")
	cmd.PersistentFlags().StringP("attr", "a", "",
		"Extract an attribute value instead of node content for provided CSS query")
	cmd.PersistentFlags().BoolP("node", "n", utils.GetConfig().Node,
		"Return the node content instead of text")
	cmd.PersistentFlags().BoolP("json", "j", false, "Output the result as JSON")
	cmd.PersistentFlags().Bool("compact", false, "Compact JSON output (no indentation)")
	cmd.PersistentFlags().IntP("depth", "d", -1, "Maximum nesting depth for JSON output (-1 for unlimited)")
	cmd.PersistentFlags().BoolP("in-place", "i", false, "Format file in place")
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

func getXpathQuery(flags *pflag.FlagSet) (query string, single bool) {
	if query, _ = flags.GetString("xpath"); query != "" {
		return query, false
	}

	query, _ = flags.GetString("extract")
	return query, true
}

func getColorMode(flags *pflag.FlagSet) int {
	inPlace, _ := flags.GetBool("in-place")
	if inPlace {
		return utils.ColorsDisabled
	}

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

func processAsJSON(flags *pflag.FlagSet, reader io.Reader, w io.Writer, contentType utils.ContentType) error {
	var (
		jsonCompact bool
		jsonDepth   int
		result      interface{}
	)
	jsonCompact, _ = flags.GetBool("compact")
	if flags.Changed("depth") {
		jsonDepth, _ = flags.GetInt("depth")
	} else {
		jsonDepth = -1
	}

	switch contentType {
	case utils.ContentXml, utils.ContentHtml:
		doc, err := xmlquery.Parse(reader)
		if err != nil {
			return fmt.Errorf("error while parsing XML: %w", err)
		}
		result = utils.NodeToJSON(doc, jsonDepth)
	case utils.ContentJson:
		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(&result); err != nil {
			return fmt.Errorf("error while parsing JSON: %w", err)
		}
	default:
		// Treat as plain text
		content, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("error while reading content: %w", err)
		}
		result = map[string]interface{}{
			"text": string(content),
		}
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error while marshaling JSON: %w", err)
	}
	indent := ""
	if !jsonCompact {
		indent = "  "
	}
	colors := getColorMode(flags)
	return utils.FormatJson(bytes.NewReader(jsonData), w, indent, colors)
}
