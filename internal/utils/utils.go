package utils

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/fatih/color"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	ColorsDefault = iota
	ColorsForced
	ColorsDisabled
)

type ContentType int

const (
	ContentXml ContentType = iota
	ContentHtml
	ContentJson
	ContentText
)

type QueryOptions struct {
	WithTags bool
	Indent   string
	Colors   int
}

const (
	jsonTokenTopValue = iota
	jsonTokenArrayStart
	jsonTokenArrayValue
	jsonTokenArrayComma
	jsonTokenObjectStart
	jsonTokenObjectKey
	jsonTokenObjectColon
	jsonTokenObjectValue
	jsonTokenObjectComma
)

func FormatXml(reader io.Reader, writer io.Writer, indent string, colors int) error {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.CharsetReader = getCharsetReader

	level := 0
	hasContent := false
	nsAliases := map[string]string{"http://www.w3.org/XML/1998/namespace": "xml"}
	lastTagName := ""
	startTagClosed := true
	newline := "\n"
	if indent == "" {
		newline = ""
	}

	if ColorsDefault != colors {
		color.NoColor = colors == ColorsDisabled
	}

	tagColor := color.New(color.FgYellow).SprintFunc()
	attrColor := color.New(color.FgGreen).SprintFunc()
	commentColor := color.New(color.FgHiBlue).SprintFunc()

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch typedToken := token.(type) {
		case xml.ProcInst:
			_, _ = fmt.Fprintf(writer, "%s%s", tagColor("<?"), typedToken.Target)

			pi := strings.TrimSpace(string(typedToken.Inst))
			attrs := strings.Split(pi, " ")
			for _, attr := range attrs {
				attrComponents := strings.SplitN(attr, "=", 2)
				_, _ = fmt.Fprintf(writer, " %s%s", attrComponents[0], attrColor("="+attrComponents[1]))
			}

			_, _ = fmt.Fprint(writer, tagColor("?>"), newline)
		case xml.StartElement:
			if !startTagClosed {
				_, _ = fmt.Fprint(writer, tagColor(">"))
				startTagClosed = true
			}
			if level > 0 {
				_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level))
			}
			var attrs []string
			for _, attr := range typedToken.Attr {
				if attr.Name.Space == "xmlns" && nsAliases[attr.Value] == "" {
					nsAliases[attr.Value] = attr.Name.Local
				}
				if attr.Name.Local == "xmlns" {
					nsAliases[attr.Value] = ""
				}
				escapedValue, _ := escapeText(attr.Value)
				attrElement := getTokenFullName(attr.Name, nsAliases) + attrColor("=\""+escapedValue+"\"")
				attrs = append(attrs, attrElement)
			}
			attrsStr := strings.Join(attrs, " ")
			if attrsStr != "" {
				attrsStr = " " + attrsStr
			}
			currentTagName := getTokenFullName(typedToken.Name, nsAliases)
			_, _ = fmt.Fprint(writer, tagColor("<"+currentTagName)+attrsStr)
			lastTagName = currentTagName
			startTagClosed = false
			level++
			hasContent = false
		case xml.CharData:
			str := normalizeSpaces(string(typedToken), indent, level)
			hasContent = str != ""
			if hasContent && !startTagClosed {
				_, _ = fmt.Fprint(writer, tagColor(">"))
				startTagClosed = true
			}
			if hasContent && (strings.Contains(str, "&") || strings.Contains(str, "<")) {
				str = "<![CDATA[" + str + "]]>"
			}
			_, _ = fmt.Fprint(writer, str)
		case xml.Comment:
			if !startTagClosed {
				_, _ = fmt.Fprint(writer, tagColor(">"))
				startTagClosed = true
			}

			for index, commentLine := range strings.Split(string(typedToken), "\n") {
				if !hasContent && level > 0 {
					_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level))
				}
				if index == 0 {
					_, _ = fmt.Fprint(writer, commentColor("<!--"))
				}
				_, _ = fmt.Fprint(writer, commentColor(commentLine))
			}
			_, _ = fmt.Fprint(writer, commentColor("-->"))

			if level == 0 {
				_, _ = fmt.Fprint(writer, newline)
			}
		case xml.EndElement:
			if level > 0 {
				level--
			}
			currentTagName := getTokenFullName(typedToken.Name, nsAliases)
			if !hasContent {
				if lastTagName != currentTagName {
					if !startTagClosed {
						_, _ = fmt.Fprint(writer, tagColor(">"))
						startTagClosed = true
					}
					_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level), tagColor("</"+currentTagName+">"))
				} else {
					_, _ = fmt.Fprint(writer, tagColor("/>"))
					startTagClosed = true
				}
			} else {
				_, _ = fmt.Fprint(writer, tagColor("</"+currentTagName+">"))
			}
			hasContent = false
			lastTagName = currentTagName
			if startTagClosed {
				lastTagName = ""
			}
		case xml.Directive:
			_, _ = fmt.Fprint(writer, tagColor("<!"), string(typedToken), tagColor(">"))
			_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level))
		default:
		}
	}

	_, _ = fmt.Fprint(writer, "\n")

	return nil
}

func XPathQuery(reader io.Reader, writer io.Writer, query string, singleNode bool, options QueryOptions) (errRes error) {
	defer func() {
		if err := recover(); err != nil {
			errRes = fmt.Errorf("XPath error: %v", err)
		}
	}()

	doc, err := xmlquery.ParseWithOptions(reader, xmlquery.ParserOptions{
		Decoder: &xmlquery.DecoderOptions{
			Strict:        false,
			CharsetReader: getCharsetReader,
		},
	})
	if err != nil {
		return err
	}

	if singleNode {
		if n := xmlquery.FindOne(doc, query); n != nil {
			return printNodeContent(writer, n, options)
		}
	} else if options.WithTags {
		for _, n := range xmlquery.Find(doc, query) {
			err := printNodeContent(writer, n, options)
			if err != nil {
				return err
			}
		}
	} else {
		expr, _ := xpath.Compile(query)
		if expr == nil {
			return errors.New("unable to parse the XPath query")
		}

		val := expr.Evaluate(xmlquery.CreateXPathNavigator(doc))

		switch typedVal := val.(type) {
		case float64:
			_, err = fmt.Fprintf(writer, "%.0f\n", typedVal)
		case string:
			_, err = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(typedVal))
		case *xpath.NodeIterator:
			for typedVal.MoveNext() {
				typedVal.Current()
				_, err = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(typedVal.Current().Value()))
				if err != nil {
					break
				}
			}
		default:
			return fmt.Errorf("unknown type error: %v", val)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func printNodeContent(writer io.Writer, node *xmlquery.Node, options QueryOptions) error {
	if options.WithTags {
		reader := strings.NewReader(node.OutputXML(true))
		return FormatXml(reader, writer, options.Indent, options.Colors)
	}

	_, err := fmt.Fprintf(writer, "%s\n", strings.TrimSpace(node.InnerText()))
	return err
}

func CSSQuery(reader io.Reader, writer io.Writer, query string, attr string, options QueryOptions) error {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}

	doc.Find(query).Each(func(index int, item *goquery.Selection) {
		if attr != "" {
			_, _ = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(item.AttrOr(attr, "")))
		} else {
			if options.WithTags {
				node := item.Nodes[0]
				tagName := node.Data
				var attrs []string
				attrsStr := ""
				for _, tagAttr := range node.Attr {
					escapedValue, _ := escapeText(tagAttr.Val)
					attrs = append(attrs, tagAttr.Key+"=\""+escapedValue+"\"")
				}
				if len(attrs) > 0 {
					attrsStr = " " + strings.Join(attrs, " ")
				}
				html, _ := item.Html()
				reader := strings.NewReader(fmt.Sprintf("<%s%s>%s</%s>", tagName, attrsStr, html, tagName))
				FormatHtml(reader, writer, options.Indent, options.Colors)
			} else {
				_, _ = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(item.Text()))
			}
		}
	})

	return nil
}

func FormatHtml(reader io.Reader, writer io.Writer, indent string, colors int) error {
	tokenizer := html.NewTokenizer(reader)

	if ColorsDefault != colors {
		color.NoColor = colors == ColorsDisabled
	}

	tagColor := color.New(color.FgYellow).SprintFunc()
	attrColor := color.New(color.FgGreen).SprintFunc()
	commentColor := color.New(color.FgHiBlue).SprintFunc()

	level := 0
	hasContent := false
	forceNewLine := false
	selfClosingTags := getSelfClosingTags()
	newline := "\n"
	if indent == "" {
		newline = ""
	}

	for {
		token := tokenizer.Next()

		if token == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			return err
		}

		switch token {
		case html.TextToken:
			str := normalizeSpaces(string(tokenizer.Text()), indent, level)
			hasContent = str != ""
			if hasContent {
				str = escapeTextContent(str)
			}
			_, _ = fmt.Fprint(writer, str)
		case html.StartTagToken, html.SelfClosingTagToken:
			if level > 0 {
				_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level))
			}

			tagName, hasAttr := tokenizer.TagName()
			selfClosingTag := token == html.SelfClosingTagToken

			if !selfClosingTag && selfClosingTags[string(tagName)] {
				selfClosingTag = true
			}

			var attrs []string
			attrsStr := ""

			if hasAttr {
				for {
					attrKey, attrValue, moreAttr := tokenizer.TagAttr()
					escapedValue, _ := escapeText(string(attrValue))
					attrs = append(attrs, string(attrKey)+attrColor("=\""+escapedValue+"\""))
					if !moreAttr {
						break
					}
				}

				attrsStr = " " + strings.Join(attrs, " ")
			}

			_, _ = fmt.Fprint(writer, tagColor("<"+string(tagName))+attrsStr)

			if selfClosingTag {
				_, _ = fmt.Fprint(writer, tagColor("/>"))
			} else {
				level++
				_, _ = fmt.Fprint(writer, tagColor(">"))
				forceNewLine = false
			}
		case html.EndTagToken:
			if level > 0 {
				level--
			}
			tagName, _ := tokenizer.TagName()

			if forceNewLine {
				_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level))
			}
			_, _ = fmt.Fprint(writer, tagColor("</"+string(tagName)+">"))

			hasContent = false
			forceNewLine = true
		case html.DoctypeToken:
			docType := tokenizer.Text()
			_, _ = fmt.Fprint(writer, tagColor("<!doctype "), string(docType), tagColor(">"), newline)
		case html.CommentToken:
			for _, commentLine := range strings.Split(string(tokenizer.Raw()), "\n") {
				if !hasContent && level > 0 {
					_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level))
				}
				_, _ = fmt.Fprint(writer, commentColor(commentLine))
			}

			if level == 0 {
				_, _ = fmt.Fprint(writer, newline)
			}
		}
	}

	_, _ = fmt.Fprint(writer, "\n")

	return nil
}

func FormatJson(reader io.Reader, writer io.Writer, indent string, colors int) error {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()

	if ColorsDefault != colors {
		color.NoColor = colors == ColorsDisabled
	}

	tagColor := color.New(color.FgYellow).SprintFunc()
	attrColor := color.New(color.FgHiBlue).SprintFunc()
	valueColor := color.New(color.FgGreen).SprintFunc()

	level := 0
	suffix := ""
	prefix := ""
	newline := "\n"
	if indent == "" {
		newline = ""
	}

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		v := reflect.ValueOf(*decoder)
		tokenState := v.FieldByName("tokenState").Int()

		switch tokenType := token.(type) {
		case json.Delim:
			switch rune(tokenType) {
			case '{':
				_, _ = fmt.Fprint(writer, prefix, tagColor("{"), newline)
				level++
				suffix = strings.Repeat(indent, level)
			case '}':
				if level > 0 {
					level--
				}
				_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level), tagColor("}"))
				if tokenState == jsonTokenArrayComma {
					suffix = "," + newline + strings.Repeat(indent, level)
				}
			case '[':
				_, _ = fmt.Fprint(writer, prefix, tagColor("["), newline)
				level++
				suffix = strings.Repeat(indent, level)
			case ']':
				if level > 0 {
					level--
				}
				_, _ = fmt.Fprint(writer, newline, strings.Repeat(indent, level), tagColor("]"))
			}
		case string:
			escapedToken := strconv.Quote(token.(string))
			value := valueColor(escapedToken)
			if tokenState == jsonTokenObjectColon {
				value = attrColor(escapedToken)
			}
			_, _ = fmt.Fprintf(writer, "%s%s", prefix, value)
		case float64:
			_, _ = fmt.Fprintf(writer, "%s%v", prefix, valueColor(token))
		case json.Number:
			_, _ = fmt.Fprintf(writer, "%s%v", prefix, valueColor(token))
		case bool:
			_, _ = fmt.Fprintf(writer, "%s%v", prefix, valueColor(token))
		case nil:
			_, _ = fmt.Fprintf(writer, "%s%s", prefix, valueColor("null"))
		}

		switch tokenState {
		case jsonTokenObjectColon:
			suffix = ": "
		case jsonTokenObjectComma:
			suffix = "," + newline + strings.Repeat(indent, level)
		case jsonTokenArrayComma:
			suffix = "," + newline + strings.Repeat(indent, level)
		}

		prefix = suffix
	}

	_, _ = fmt.Fprint(writer, "\n")

	return nil
}

func IsHTML(input string) bool {
	input = strings.ToLower(input)
	htmlMarkers := []string{"html", "<!d", "<body"}

	for _, htmlMarker := range htmlMarkers {
		if strings.Contains(input, htmlMarker) {
			return true
		}
	}

	return false
}

func IsJSON(input string) bool {
	input = strings.ToLower(input)
	matched, _ := regexp.MatchString(`\s*[{\[]`, input)
	return matched
}

func PagerPrint(reader io.Reader, writer io.Writer) error {
	pager := os.Getenv("PAGER")

	if pager != "less" {
		_, err := io.Copy(writer, reader)
		return err
	}

	cmd := exec.Command(pager, "--quit-if-one-screen", "--no-init", "--RAW-CONTROL-CHARS")
	cmd.Stdin = reader
	cmd.Stdout = writer

	return cmd.Run()
}

func getTokenFullName(name xml.Name, nsAliases map[string]string) string {
	result := name.Local
	if name.Space != "" {
		space := name.Space
		if alias, ok := nsAliases[space]; ok {
			space = alias
		}
		if space != "" {
			result = space + ":" + name.Local
		}
	}
	return result
}

func getSelfClosingTags() map[string]bool {
	return map[string]bool{
		"area":   true,
		"base":   true,
		"br":     true,
		"col":    true,
		"embed":  true,
		"hr":     true,
		"img":    true,
		"input":  true,
		"keygen": true,
		"link":   true,
		"meta":   true,
		"param":  true,
		"source": true,
		"track":  true,
		"wbr":    true,
	}
}

func escapeText(input string) (string, error) {
	buf := new(bytes.Buffer)
	if err := xml.EscapeText(buf, []byte(input)); err != nil {
		return "", err
	}

	result := buf.String()
	result = strings.Replace(result, "&#34;", "&quot;", -1)
	result = strings.Replace(result, "&#39;", "&apos;", -1)

	return result, nil
}

func escapeTextContent(input string) string {
	// Only escape the minimal set of characters needed for text content
	// to avoid XML parsing errors: & < >
	result := strings.ReplaceAll(input, "&", "&amp;")
	result = strings.ReplaceAll(result, "<", "&lt;")
	result = strings.ReplaceAll(result, ">", "&gt;")
	return result
}

func normalizeSpaces(input string, indent string, level int) string {
	if strings.TrimSpace(input) == "" {
		input = ""
	}

	regexpHead, _ := regexp.Compile("^ *\n +")
	if regexpHead.MatchString(input) {
		input = strings.TrimLeft(input, " \n")
		input = "\n" + strings.Repeat(indent, level) + input
	}

	regexpTail, _ := regexp.Compile("\n +$")
	if regexpTail.MatchString(input) {
		input = strings.TrimRight(input, " \n")
		input += "\n" + strings.Repeat(indent, level-1)
	} else {
		input = strings.TrimRight(input, " ")
	}

	return input
}

func getCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	if strings.ToLower(charset) == "utf-16" {
		charset = "utf-8"
	}
	e, err := ianaindex.MIME.Encoding(charset)
	if err != nil {
		return nil, err
	}
	return transform.NewReader(input, e.NewDecoder()), nil
}
