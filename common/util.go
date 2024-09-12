package common

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func Text(content string) string {
	// 保留指定的HTML标签
	doc, _ := html.Parse(strings.NewReader(content))
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data != "p" && n.Data != "a" && n.Data != "code" {
				n.Parent.RemoveChild(n)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	content, _ = renderNode(doc)

	// 替换特定字符串
	replacements := map[string]string{
		"<p>":      "",
		"</p>":     "",
		"&nbsp;":   "",
		"&ldquo;":  "",
		"&rdquo;":  "",
		"&hellip;": "",
		"&mdash":   "",
	}
	for old, new := range replacements {
		content = strings.ReplaceAll(content, old, new)
	}

	// 替换特定字符为换行符
	content = strings.ReplaceAll(content, "&nbsp;", "\n")
	content = strings.ReplaceAll(content, "</br>", "\n")

	// 移除p标签的style属性
	re := regexp.MustCompile(`<p(.*?)style="(.*?)"(.*?)>`)
	content = re.ReplaceAllString(content, "<p$1$3>")

	return content
}

// 辅助函数：将HTML节点渲染为字符串
func renderNode(n *html.Node) (string, error) {
	var buf strings.Builder
	err := html.Render(&buf, n)
	return buf.String(), err
}
