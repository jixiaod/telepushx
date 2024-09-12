package common

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func Text(content string) string {

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return content // 如果解析失败，返回原始字符串
	}

	var buf strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p":
				buf.WriteString("\n") // 在<p>标签前添加换行
			case "br":
				buf.WriteString("\n")
			}
		} else if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
		if n.Type == html.ElementNode && n.Data == "p" {
			buf.WriteString("\n") // 在</p>标签后添加换行
		}
	}
	f(doc)

	result := buf.String()

	// 替换特定字符串
	replacements := map[string]string{
		"&nbsp;":   " ",
		"&ldquo;":  "\"",
		"&rdquo;":  "\"",
		"&hellip;": "…",
		"&mdash":   "—",
	}
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// 移除多余的空行
	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")

	// 移除首尾空白字符
	result = strings.TrimSpace(result)

	return result
}
