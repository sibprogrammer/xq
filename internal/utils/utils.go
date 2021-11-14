package utils

import (
	"encoding/xml"
	"fmt"
	"github.com/antchfx/xmlquery"
	"github.com/fatih/color"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func FormatXml(str string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(str))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		e, err := ianaindex.MIME.Encoding(charset)
		if err != nil {
			return nil, err
		}
		return transform.NewReader(input, e.NewDecoder()), nil
	}

	level := 0
	hasContent := false
	result := new(strings.Builder)
	nsAliases := map[string]string{}
	lastTagName := ""

	tagColor := color.New(color.FgYellow).SprintFunc()
	attrColor := color.New(color.FgGreen).SprintFunc()
	commentColor := color.New(color.FgHiBlue).SprintFunc()

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}

		switch typedToken := token.(type) {
		case xml.ProcInst:
			_, _ = fmt.Fprintf(result, "%s%s %s%s\n", tagColor("<?"), typedToken.Target, string(typedToken.Inst), tagColor("?>"))
		case xml.StartElement:
			if level > 0 {
				_, _ = fmt.Fprint(result, "\n", strings.Repeat("  ", level))
			}
			var attrs []string
			for _, attr := range typedToken.Attr {
				if attr.Name.Space == "xmlns" {
					nsAliases[attr.Value] = attr.Name.Local
				}
				attrs = append(attrs, getTokenFullName(attr.Name, nsAliases) + attrColor("=\"" + attr.Value + "\""))
			}
			attrsStr := strings.Join(attrs, " ")
			if attrsStr != "" {
				attrsStr = " " + attrsStr
			}
			currentTagName := getTokenFullName(typedToken.Name, nsAliases)
			_, _ = fmt.Fprint(result, tagColor("<" + currentTagName) + attrsStr + tagColor(">"))
			lastTagName = currentTagName
			level++
		case xml.CharData:
			str := string(typedToken)
			str = strings.TrimSpace(str)
			_, _ = fmt.Fprint(result, str)
			hasContent = str != ""
		case xml.Comment:
			if !hasContent && level > 0 {
				_, _ = fmt.Fprint(result, "\n", strings.Repeat("  ", level))
			}
			_, _ = fmt.Fprint(result, commentColor("<!--" + string(typedToken) + "-->"))
			if level == 0 {
				_, _ = fmt.Fprint(result, "\n")
			}
		case xml.EndElement:
			level--
			currentTagName := getTokenFullName(typedToken.Name, nsAliases)
			if !hasContent {
				if lastTagName != currentTagName {
					_, _ = fmt.Fprint(result, "\n", strings.Repeat("  ", level), tagColor("</"+currentTagName+">"))
				} else {
					str := result.String()
					result.Reset()
					result.WriteString(str[:len(str)-len(tagColor(">"))])
					_, _ = fmt.Fprint(result, tagColor("/>"))
				}
			} else {
				_, _ = fmt.Fprint(result, tagColor("</"+currentTagName+">"))
			}
			hasContent = false
			lastTagName = currentTagName
		default:
		}
	}

	return result.String(), nil
}

func XPathQuery(str string, query string) (string, error) {
	result := new(strings.Builder)

	doc, err := xmlquery.Parse(strings.NewReader(str))
	if err != nil {
		return "", err
	}

	for _, n := range xmlquery.Find(doc, query) {
		_, _ = fmt.Fprintf(result, "%s\n", n.InnerText())
	}

	return result.String(), nil
}

func PagerPrint(str string) {
	pager := os.Getenv("PAGER")

	if pager != "less" {
		fmt.Println(str)
		return
	}

	cmd := exec.Command(pager, "--quit-if-one-screen", "--no-init")
	cmd.Stdin = strings.NewReader(str)
	cmd.Stdout = os.Stdout

	err := cmd.Run()

	if err != nil {
		log.Fatal("Failed to run the pager:", err)
	}
}

func getTokenFullName(name xml.Name, nsAliases map[string]string) string  {
	result := name.Local
	if name.Space != "" {
		space := name.Space
		if alias, ok := nsAliases[space]; ok {
			space = alias
		}
		result = space + ":" + name.Local
	}
	return result
}