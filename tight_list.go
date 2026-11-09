package main

import (
	"regexp"
	"strings"
)

var listItem = regexp.MustCompile(`^\s*([-*+]|\d+[.)])\s+\S`)

// 項目の間に空行があると、Markdown は項目の文を <p> で包む（loose list）。
// 入れ子を持つ項目では親の文と子のリストの間に段落の余白が入るので、
// 前後がどちらも項目の行のときだけ空行を消す。
// 項目の中に2段落目やコードを持つリストは前の行が項目にならないので触らない。
func tightenLists(lines []string) []string {
	out := make([]string, 0, len(lines))
	fence := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence == "" {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				fence = trimmed[:3]
			}
		} else if strings.HasPrefix(trimmed, fence) {
			fence = ""
		}
		if fence == "" && trimmed == "" && 0 < i && i+1 < len(lines) &&
			listItem.MatchString(lines[i-1]) && listItem.MatchString(lines[i+1]) {
			continue
		}
		out = append(out, line)
	}
	return out
}
