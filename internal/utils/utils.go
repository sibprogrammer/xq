package utils

import (
	"encoding/xml"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/xmlquery"
	"github.com/fatih/color"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	ColorsDefault = iota
	ColorsForced
	ColorsDisabled
)

func FormatXml(reader io.Reader, writer io.Writer, indent string, colors int) error {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		e, err := ianaindex.MIME.Encoding(charset)
		if err != nil {
			return nil, err
		}
		return transform.NewReader(input, e.NewDecoder()), nil
	}

	level := 0
	hasContent := false
	nsAliases := map[string]string{}
	lastTagName := ""
	startTagClosed := true

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

			_, _ = fmt.Fprintf(writer, "%s\n", tagColor("?>"))
		case xml.StartElement:
			if !startTagClosed {
				_, _ = fmt.Fprint(writer, tagColor(">"))
				startTagClosed = true
			}
			if level > 0 {
				_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level))
			}
			var attrs []string
			for _, attr := range typedToken.Attr {
				if attr.Name.Space == "xmlns" {
					nsAliases[attr.Value] = attr.Name.Local
				}
				if attr.Name.Local == "xmlns" {
					nsAliases[attr.Value] = ""
				}
				attrs = append(attrs, getTokenFullName(attr.Name, nsAliases)+attrColor("=\""+attr.Value+"\""))
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
		case xml.CharData:
			str := string(typedToken)
			if strings.TrimSpace(str) == "" {
				str = ""
			}
			hasContent = str != ""
			if hasContent && !startTagClosed {
				_, _ = fmt.Fprint(writer, tagColor(">"))
				startTagClosed = true
			}
			_, _ = fmt.Fprint(writer, str)
		case xml.Comment:
			if !startTagClosed {
				_, _ = fmt.Fprint(writer, tagColor(">"))
				startTagClosed = true
			}
			if !hasContent && level > 0 {
				_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level))
			}
			_, _ = fmt.Fprint(writer, commentColor("<!--"+string(typedToken)+"-->"))
			if level == 0 {
				_, _ = fmt.Fprint(writer, "\n")
			}
		case xml.EndElement:
			level--
			currentTagName := getTokenFullName(typedToken.Name, nsAliases)
			if !hasContent {
				if lastTagName != currentTagName {
					if !startTagClosed {
						_, _ = fmt.Fprint(writer, tagColor(">"))
						startTagClosed = true
					}
					_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level), tagColor("</"+currentTagName+">"))
				} else {
					_, _ = fmt.Fprint(writer, tagColor("/>"))
					startTagClosed = true
				}
			} else {
				_, _ = fmt.Fprint(writer, tagColor("</"+currentTagName+">"))
			}
			hasContent = false
			lastTagName = currentTagName
		default:
		}
	}

	_, _ = fmt.Fprint(writer, "\n")

	return nil
}

func XPathQuery(reader io.Reader, writer io.Writer, query string, singleNode bool) error {
	doc, err := xmlquery.ParseWithOptions(reader, xmlquery.ParserOptions{
		Decoder: &xmlquery.DecoderOptions{
			Strict:    false,
			AutoClose: xml.HTMLAutoClose,
		},
	})
	if err != nil {
		return err
	}

	if singleNode {
		if n := xmlquery.FindOne(doc, query); n != nil {
			_, _ = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(n.InnerText()))
		}
	} else {
		for _, n := range xmlquery.Find(doc, query) {
			_, _ = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(n.InnerText()))
		}
	}

	return nil
}

func CSSQuery(reader io.Reader, writer io.Writer, query string) error {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}

	doc.Find(query).Each(func(index int, item *goquery.Selection) {
		_, _ = fmt.Fprintf(writer, "%s\n", strings.TrimSpace(item.Text()))
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
			str := string(tokenizer.Text())
			if strings.TrimSpace(str) == "" {
				str = ""
			}
			hasContent = str != ""
			_, _ = fmt.Fprint(writer, str)
		case html.StartTagToken, html.SelfClosingTagToken:
			if level > 0 {
				_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level))
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
					attrs = append(attrs, string(attrKey)+attrColor("=\""+string(attrValue)+"\""))
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
			level--
			tagName, _ := tokenizer.TagName()

			if forceNewLine {
				_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level))
			}
			_, _ = fmt.Fprint(writer, tagColor("</"+string(tagName)+">"))

			hasContent = false
			forceNewLine = true
		case html.DoctypeToken:
			docType := tokenizer.Text()
			_, _ = fmt.Fprintf(writer, "%s%s%s\n", tagColor("<!doctype "), string(docType), tagColor(">"))
		case html.CommentToken:
			comment := tokenizer.Raw()
			if !hasContent && level > 0 {
				_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level))
			}
			_, _ = fmt.Fprint(writer, commentColor(string(comment)))
			if level == 0 {
				_, _ = fmt.Fprint(writer, "\n")
			}
		}
	}

	_, _ = fmt.Fprint(writer, "\n")

	return nil
}

func PagerPrint(reader io.Reader) error {
	pager := os.Getenv("PAGER")

	if pager != "less" {
		_, err := io.Copy(os.Stdout, reader)
		return err
	}

	cmd := exec.Command(pager, "--quit-if-one-screen", "--no-init", "--RAW-CONTROL-CHARS")
	cmd.Stdin = reader
	cmd.Stdout = os.Stdout

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
