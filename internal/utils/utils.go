package utils

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/fatih/color"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

const (
	ColorsDefault = iota
	ColorsForced
	ColorsDisabled
)

const attrsLineLengthLimit = 80

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

var ErrNoMatch = errors.New("no matches found")

var xmlDeclVersion = regexp.MustCompile(`^(<\?xml\s+version\s*=\s*["'])1\.1(["'])`)

func SanitizeXmlVersion(reader io.Reader) (io.Reader, string) {
	buf := make([]byte, 64)
	length, _ := io.ReadFull(reader, buf)
	prefix := buf[:length]

	version := ""
	if index := bytes.IndexByte(prefix, '<'); index != -1 && xmlDeclVersion.Match(prefix[index:]) {
		version = "1.1"
		prefix = append(prefix[:index:index], xmlDeclVersion.ReplaceAll(prefix[index:], []byte("${1}1.0${2}"))...)
	}

	return io.MultiReader(bytes.NewReader(prefix), reader), version
}

func FormatXml(reader io.Reader, writer io.Writer, indent string, colors int) error {
	sanitizedReader, origXmlVersion := SanitizeXmlVersion(reader)
	decoder := xml.NewDecoder(sanitizedReader)
	decoder.Strict = false
	decoder.CharsetReader = getCharsetReader

	level := 0
	hasContent := false
	spaceContent := ""
	scopes := newNsScopes()
	lastTagName := ""
	startTagClosed := true
	multilineTag := false
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

	write := func(args ...any) error {
		_, err := fmt.Fprint(writer, args...)
		return err
	}

	closeTag := func(bracket string, depth int) {
		if multilineTag {
			_ = write(newline, strings.Repeat(indent, depth))
			multilineTag = false
		}
		_ = write(tagColor(bracket))
		startTagClosed = true
	}

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
			_ = write(tagColor("<?"), typedToken.Target)

			pi := strings.TrimSpace(string(typedToken.Inst))
			if typedToken.Target == "xml" && origXmlVersion != "" {
				pi = strings.Replace(pi, "1.0", origXmlVersion, 1)
				origXmlVersion = ""
			}
			if pi != "" {
				attrs := strings.Split(pi, " ")
				for _, attr := range attrs {
					attrComponents := strings.SplitN(attr, "=", 2)
					if len(attrComponents) == 2 {
						_ = write(" ", attrComponents[0], attrColor("="+attrComponents[1]))
					} else {
						_ = write(" ", attrComponents[0])
					}
				}
			}

			_ = write(tagColor("?>"), newline)
		case xml.StartElement:
			spaceContent = ""
			if !startTagClosed {
				closeTag(">", level-1)
			}
			if level > 0 {
				_ = write(newline, strings.Repeat(indent, level))
			}
			var attrs []string
			attrsLength := 0
			// A name may use a prefix declared later in its tag, so push the declarations first.
			var bindings []nsBinding
			for _, attr := range typedToken.Attr {
				if attr.Name.Space == "xmlns" {
					bindings = append(bindings, nsBinding{prefix: attr.Name.Local, uri: attr.Value})
				} else if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
					bindings = append(bindings, nsBinding{prefix: "", uri: attr.Value})
				}
			}
			scopes.push(bindings)
			for _, attr := range typedToken.Attr {
				escapedValue, _ := escapeText(attr.Value)
				attrName := scopes.attrName(attr.Name)
				attrs = append(attrs, attrName+attrColor("=\""+escapedValue+"\""))
				attrsLength += utf8.RuneCountInString(attrName) + utf8.RuneCountInString(escapedValue) + len(`="" `)
			}
			currentTagName := scopes.elementName(typedToken.Name)
			if attrsLength > attrsLineLengthLimit && newline != "" {
				_ = write(tagColor("<" + currentTagName))
				for _, attr := range attrs {
					_ = write(newline, strings.Repeat(indent, level+1), attr)
				}
				multilineTag = true
			} else {
				attrsStr := strings.Join(attrs, " ")
				if attrsStr != "" {
					attrsStr = " " + attrsStr
				}
				_ = write(tagColor("<"+currentTagName), attrsStr)
			}
			lastTagName = currentTagName
			startTagClosed = false
			level++
			hasContent = false
		case xml.CharData:
			chars := string(typedToken)
			str := normalizeSpaces(chars, indent, level)
			spaceContent = ""
			if str == "" && chars != "" && !strings.Contains(chars, "\n") && !startTagClosed {
				spaceContent = chars
			}
			hasContent = str != ""
			if hasContent && !startTagClosed {
				closeTag(">", level-1)
			}
			if hasContent && (strings.Contains(str, "&") || strings.Contains(str, "<")) {
				str = "<![CDATA[" + str + "]]>"
			}
			_ = write(str)
		case xml.Comment:
			spaceContent = ""
			if !startTagClosed {
				closeTag(">", level-1)
			}

			for index, commentLine := range strings.Split(string(typedToken), "\n") {
				if !hasContent && level > 0 {
					_ = write(newline, strings.Repeat(indent, level))
				}
				if index == 0 {
					_ = write(commentColor("<!--"))
				}
				_ = write(commentColor(commentLine))
			}
			_ = write(commentColor("-->"))

			if level == 0 {
				_ = write(newline)
			}
		case xml.EndElement:
			if level > 0 {
				level--
			}
			currentTagName := scopes.elementName(typedToken.Name)
			scopes.pop()
			if !hasContent {
				if lastTagName != currentTagName {
					if !startTagClosed {
						closeTag(">", level)
					}
					_ = write(newline, strings.Repeat(indent, level), tagColor("</"+currentTagName+">"))
				} else if spaceContent != "" {
					closeTag(">", level)
					_ = write(spaceContent, tagColor("</"+currentTagName+">"))
				} else {
					closeTag("/>", level)
				}
			} else {
				_ = write(tagColor("</" + currentTagName + ">"))
			}
			spaceContent = ""
			hasContent = false
			lastTagName = currentTagName
			if startTagClosed {
				lastTagName = ""
			}
		case xml.Directive:
			spaceContent = ""
			_ = write(tagColor("<!"), string(typedToken), tagColor(">"))
			_ = write(newline, strings.Repeat(indent, level))
		default:
		}
	}

	return write("\n")
}

func formatXPathNumber(number float64) string {
	switch {
	case math.IsNaN(number):
		return "NaN"
	case math.IsInf(number, 1):
		return "Infinity"
	case math.IsInf(number, -1):
		return "-Infinity"
	case number == 0:
		return "0"
	default:
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
}

func XPathQuery(reader io.Reader, writer io.Writer, query string, singleNode bool, options QueryOptions) (errRes error) {
	defer func() {
		if err := recover(); err != nil {
			errRes = fmt.Errorf("XPath error: %v", err)
		}
	}()

	sanitizedReader, _ := SanitizeXmlVersion(reader)
	doc, err := xmlquery.ParseWithOptions(sanitizedReader, xmlquery.ParserOptions{
		Decoder: &xmlquery.DecoderOptions{
			Strict:        false,
			CharsetReader: getCharsetReader,
		},
	})
	if err != nil {
		return err
	}

	if singleNode {
		n := xmlquery.FindOne(doc, query)
		if n == nil {
			return ErrNoMatch
		}
		return printNodeContent(writer, n, options)
	} else if options.WithTags {
		nodes := xmlquery.Find(doc, query)
		if len(nodes) == 0 {
			return ErrNoMatch
		}
		for _, n := range nodes {
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
		found := true

		switch typedVal := val.(type) {
		case float64:
			_, err = fmt.Fprintf(writer, "%s\n", formatXPathNumber(typedVal))
		case bool:
			found = typedVal
			_, err = fmt.Fprintf(writer, "%t\n", typedVal)
		case string:
			found = strings.TrimSpace(typedVal) != ""
			_, err = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(typedVal))
		case *xpath.NodeIterator:
			found = false
			for typedVal.MoveNext() {
				found = true
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

		if !found {
			return ErrNoMatch
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

	selection := doc.Find(query)
	if selection.Length() == 0 {
		return ErrNoMatch
	}

	selection.Each(func(index int, item *goquery.Selection) {
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
	tagJustOpened := false
	spaceContent := ""
	forceNewLine := false
	selfClosingTags := getSelfClosingTags()
	newline := "\n"
	if indent == "" {
		newline = ""
	}

	write := func(args ...any) error {
		_, err := fmt.Fprint(writer, args...)
		return err
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
			chars := string(tokenizer.Text())
			str := normalizeSpaces(chars, indent, level)
			spaceContent = ""
			if str == "" && chars != "" && !strings.Contains(chars, "\n") && tagJustOpened {
				spaceContent = chars
			}
			hasContent = str != ""
			if hasContent {
				str, _ = escapeText(str)
			}
			_ = write(str)
		case html.StartTagToken, html.SelfClosingTagToken:
			if level > 0 {
				_ = write(newline, strings.Repeat(indent, level))
			}

			tagName, hasAttr := tokenizer.TagName()
			selfClosingTag := token == html.SelfClosingTagToken

			if !selfClosingTag && selfClosingTags[string(tagName)] {
				selfClosingTag = true
			}

			var attrs []string
			attrsStr := ""
			attrsLength := 0

			if hasAttr {
				for {
					attrKey, attrValue, moreAttr := tokenizer.TagAttr()
					escapedValue, _ := escapeText(string(attrValue))
					attrs = append(attrs, string(attrKey)+attrColor("=\""+escapedValue+"\""))
					attrsLength += utf8.RuneCount(attrKey) + utf8.RuneCountInString(escapedValue) + len(`="" `)
					if !moreAttr {
						break
					}
				}

				attrsStr = " " + strings.Join(attrs, " ")
			}

			if attrsLength > attrsLineLengthLimit && newline != "" {
				_ = write(tagColor("<" + string(tagName)))
				for _, attr := range attrs {
					_ = write(newline, strings.Repeat(indent, level+1), attr)
				}
				_ = write(newline, strings.Repeat(indent, level))
			} else {
				_ = write(tagColor("<"+string(tagName)), attrsStr)
			}

			spaceContent = ""
			tagJustOpened = false
			if selfClosingTag {
				_ = write(tagColor("/>"))
			} else {
				level++
				_ = write(tagColor(">"))
				forceNewLine = false
				tagJustOpened = true
			}
		case html.EndTagToken:
			if level > 0 {
				level--
			}
			tagName, _ := tokenizer.TagName()

			if forceNewLine {
				_ = write(newline, strings.Repeat(indent, level))
			} else if spaceContent != "" {
				_ = write(spaceContent)
			}
			_ = write(tagColor("</" + string(tagName) + ">"))

			hasContent = false
			forceNewLine = true
			tagJustOpened = false
			spaceContent = ""
		case html.DoctypeToken:
			docType := tokenizer.Text()
			_ = write(tagColor("<!doctype "), string(docType), tagColor(">"), newline)
		case html.CommentToken:
			spaceContent = ""
			tagJustOpened = false
			for _, commentLine := range strings.Split(string(tokenizer.Raw()), "\n") {
				if !hasContent && level > 0 {
					_ = write(newline, strings.Repeat(indent, level))
				}
				_ = write(commentColor(commentLine))
			}

			if level == 0 {
				_ = write(newline)
			}
		}
	}

	return write("\n")
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

	newline := "\n"
	if indent == "" {
		newline = ""
	}
	write := func(args ...any) error {
		_, err := fmt.Fprint(writer, args...)
		return err
	}

	var formatToken func(json.Token, int) error
	formatToken = func(token json.Token, level int) error {
		switch typedToken := token.(type) {
		case json.Delim:
			switch rune(typedToken) {
			case '{':
				if err := write(tagColor("{")); err != nil {
					return err
				}
				level++
				index := 0
				for decoder.More() {
					if index > 0 {
						if err := write(",", newline); err != nil {
							return err
						}
					} else if err := write(newline); err != nil {
						return err
					}
					if err := write(strings.Repeat(indent, level)); err != nil {
						return err
					}

					keyToken, err := decoder.Token()
					if err != nil {
						return err
					}
					key, ok := keyToken.(string)
					if !ok {
						return fmt.Errorf("expected JSON object key, got %T", keyToken)
					}

					valueToken, err := decoder.Token()
					if err != nil {
						return err
					}

					if err := write(attrColor(strconv.Quote(key)), ": "); err != nil {
						return err
					}
					if err := formatToken(valueToken, level); err != nil {
						return err
					}
					index++
				}
				level--

				endToken, err := decoder.Token()
				if err != nil {
					return err
				}
				if endToken != json.Delim('}') {
					return fmt.Errorf("expected JSON object end, got %v", endToken)
				}
				if index > 0 {
					return write(newline, strings.Repeat(indent, level), tagColor("}"))
				}
				return write(tagColor("}"))
			case '[':
				if err := write(tagColor("[")); err != nil {
					return err
				}
				level++
				index := 0
				for decoder.More() {
					if index > 0 {
						if err := write(",", newline); err != nil {
							return err
						}
					} else if err := write(newline); err != nil {
						return err
					}
					if err := write(strings.Repeat(indent, level)); err != nil {
						return err
					}

					valueToken, err := decoder.Token()
					if err != nil {
						return err
					}
					if err := formatToken(valueToken, level); err != nil {
						return err
					}
					index++
				}
				level--

				endToken, err := decoder.Token()
				if err != nil {
					return err
				}
				if endToken != json.Delim(']') {
					return fmt.Errorf("expected JSON array end, got %v", endToken)
				}
				if index > 0 {
					return write(newline, strings.Repeat(indent, level), tagColor("]"))
				}
				return write(tagColor("]"))
			default:
				return fmt.Errorf("unexpected JSON delimiter %q", typedToken)
			}
		case string:
			return write(valueColor(strconv.Quote(typedToken)))
		case float64, json.Number, bool:
			return write(valueColor(typedToken))
		case nil:
			return write(valueColor("null"))
		}

		return nil
	}

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if err := formatToken(token, 0); err != nil {
			return err
		}
	}

	return write("\n")
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
	matched, _ := regexp.MatchString(`^\s*[{\[]`, input)
	return matched
}

func PagerPrint(reader io.Reader, writer io.Writer, pager string) error {
	var args []string

	if pager == "" {
		_, err := io.Copy(writer, reader)
		return err
	}

	if pager == "less" {
		args = append(args, "--quit-if-one-screen", "--no-init", "--RAW-CONTROL-CHARS")
	}

	cmd := exec.Command(pager, args...)
	cmd.Stdin = reader
	cmd.Stdout = writer

	return cmd.Run()
}

// nsBinding is a namespace declaration: xmlns:prefix="uri", or xmlns="uri" with an empty prefix.
type nsBinding struct {
	prefix, uri string
}

// nsScopes has the namespace declarations in scope: the predeclared xml prefix, then those of
// each open element.
type nsScopes struct {
	// frames has the declarations of each scope, outermost first.
	frames [][]nsBinding
	// uris has, for each prefix, the URIs it is bound to, outermost first.
	uris map[string][]string
	// prefixes has, for each URI, the prefixes bound to it, outermost first.
	prefixes map[string][]string
}

func newNsScopes() *nsScopes {
	s := &nsScopes{uris: map[string][]string{}, prefixes: map[string][]string{}}
	s.push([]nsBinding{{prefix: "xml", uri: "http://www.w3.org/XML/1998/namespace"}})
	return s
}

// push adds the declarations of an element being opened.
func (s *nsScopes) push(bindings []nsBinding) {
	s.frames = append(s.frames, bindings)
	for _, b := range bindings {
		s.uris[b.prefix] = append(s.uris[b.prefix], b.uri)
		s.prefixes[b.uri] = append(s.prefixes[b.uri], b.prefix)
	}
}

// pop removes the declarations of the innermost open element, but never the predeclared ones.
func (s *nsScopes) pop() {
	if len(s.frames) <= 1 {
		return
	}
	bindings := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]
	for i := len(bindings) - 1; i >= 0; i-- {
		b := bindings[i]
		dropLast(s.uris, b.prefix)
		dropLast(s.prefixes, b.uri)
	}
}

// dropLast removes the last value listed under key, and the key once it has none.
func dropLast(m map[string][]string, key string) {
	if values := m[key]; len(values) > 1 {
		m[key] = values[:len(values)-1]
	} else {
		delete(m, key)
	}
}

// lookup returns the namespace URI that prefix is bound to in the current scope, or "" if none.
func (s *nsScopes) lookup(prefix string) string {
	if uris := s.uris[prefix]; len(uris) > 0 {
		return uris[len(uris)-1]
	}
	return ""
}

// elementName returns name without a prefix if it is in the default namespace, else as
// prefixedName does.
func (s *nsScopes) elementName(name xml.Name) string {
	if s.lookup("") == name.Space {
		return name.Local
	}
	return s.prefixedName(name)
}

// attrName returns name as prefixedName does, but keeps the xmlns prefix of a namespace
// declaration, which the decoder reports in namespace xmlns. A default namespace never applies to
// attributes.
func (s *nsScopes) attrName(name xml.Name) string {
	if name.Space == "xmlns" {
		return "xmlns:" + name.Local
	}
	return s.prefixedName(name)
}

// prefixedName returns name with the outermost prefix bound to its namespace, or else with the
// namespace as the prefix, which keeps an undeclared prefix: the decoder reports it as the
// namespace.
func (s *nsScopes) prefixedName(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	for _, prefix := range s.prefixes[name.Space] {
		if prefix != "" && s.lookup(prefix) == name.Space {
			return prefix + ":" + name.Local
		}
	}
	return name.Space + ":" + name.Local
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
	result = strings.Replace(result, "&#10;", "\n", -1)
	result = strings.Replace(result, "&#xA;", "\n", -1)

	return result, nil
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
