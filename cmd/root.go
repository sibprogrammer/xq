package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

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
			var indent string

			overwrite, _ := cmd.Flags().GetBool("overwrite")
			if indent, err = getIndent(cmd.Flags()); err != nil {
				return err
			}
			xPathQuery, singleNode := getXpathQuery(cmd.Flags())
			withTags, _ := cmd.Flags().GetBool("node")
			var colors int
			if overwrite {
				colors = utils.ColorsDisabled
			} else {
				colors = getColorMode(cmd.Flags())
			}

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

			var pagerPR *io.PipeReader
			var pagerPW *io.PipeWriter
			if !overwrite {
				pagerPR, pagerPW = io.Pipe()
			}

			var totalFiles int
			if len(args) == 0 {
				totalFiles = 1
			} else {
				totalFiles = len(args)
			}
			// The “totalFiles * 2” part comes from the fact that there’s two goroutines
			// inside the for loop and the fact that those two goroutines send a maximum
			// of one value to errChan. The “+ 1” part comes from the fact that there’s
			// one goroutine outside of the for loop and the fact that that goroutine
			// sends a maximum of one value to errChan.
			errChan := make(chan error, totalFiles * 2 + 1)
			var wg sync.WaitGroup
			for i := 0; i < totalFiles; i++ {
				var path string
				var reader io.Reader
				if len(args) == 0 {
					fileInfo, _ := os.Stdin.Stat()

					if (fileInfo.Mode() & os.ModeCharDevice) != 0 {
						_ = cmd.Help()
						return nil
					}

					if overwrite {
						return errors.New("--overwrite was used but no filenames were specified")
					}

					reader = os.Stdin
				} else {
					path = args[i]
					if reader, err = os.Open(path); err != nil {
						return err
					}
				}

				formattedPR, formattedPW := io.Pipe()

				wg.Add(1)
				go func(reader io.Reader, formattedPW *io.PipeWriter) {
					defer wg.Done()
					defer formattedPW.Close()

					var err error
					if xPathQuery != "" {
						err = utils.XPathQuery(reader, formattedPW, xPathQuery, singleNode, options)
					} else if cssQuery != "" {
						err = utils.CSSQuery(reader, formattedPW, cssQuery, cssAttr, options)
					} else {
						var contentType utils.ContentType
						contentType, reader = detectFormat(cmd.Flags(), reader)
						if jsonOutputMode {
							err = processAsJSON(cmd.Flags(), reader, formattedPW, contentType)
						} else {
							switch contentType {
							case utils.ContentHtml:
								err = utils.FormatHtml(reader, formattedPW, indent, colors)
							case utils.ContentXml:
								err = utils.FormatXml(reader, formattedPW, indent, colors)
							case utils.ContentJson:
								err = utils.FormatJson(reader, formattedPW, indent, colors)
							default:
								err = fmt.Errorf("unknown content type: %v", contentType)
							}
						}
					}

					errChan <- err
				}(reader, formattedPW)

				wg.Add(1)
				go func(formattedPR *io.PipeReader, path string) {
					defer wg.Done()
					defer formattedPR.Close()

					var err error
					var allData []byte
					if allData, err = io.ReadAll(formattedPR); err != nil {
						errChan <- err
						return
					}
					if overwrite {
						if err = os.WriteFile(path, allData, 0666); err != nil {
							errChan <- err
							return
						}
					} else {
						if _, err = pagerPW.Write(allData); err != nil {
							errChan <- err
							return
						}
					}
				}(formattedPR, path)
			}

			go func() {
				wg.Wait()
				if !overwrite {
					if err := pagerPW.Close(); err != nil {
						errChan <- err
					}
				}
				close(errChan)
			}()

			if !overwrite {
				if err = utils.PagerPrint(pagerPR, cmd.OutOrStdout()); err != nil {
					return err
				}
				if err = pagerPR.Close(); err != nil {
					return err
				}
			}

			for err = range errChan {
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func InitFlags(cmd *cobra.Command) {
	homeDir, _ := os.UserHomeDir()
	configFile := path.Join(homeDir, ".xq")
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
	cmd.PersistentFlags().Bool("overwrite", false, "Instead of printing the formatted file, replace the original with the formatted version")
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
		print(err.Error())
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
