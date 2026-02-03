package utils

import (
	"strings"
)

// ParseJSONP 提取 jsonpgz(...) 括号中的 JSON 内容
func ParseJSONP(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")

	// 如果找到了括号，并且顺序正确
	if start != -1 && end != -1 && start < end {
		return raw[start : end+1]
	}
	
	// 如果没找到，返回空字符串
	return ""
}