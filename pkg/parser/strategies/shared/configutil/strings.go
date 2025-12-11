// Package configutil provides shared utilities for parsing framework config files.
package configutil

import "regexp"

// QuotedStringRegex matches single or double quoted strings.
var QuotedStringRegex = regexp.MustCompile(`['"]([^'"]+)['"]`)

// ExtractQuotedStrings extracts all quoted string values from array content.
// For example, given `['a.js', 'b.js']`, it returns ["a.js", "b.js"].
func ExtractQuotedStrings(arrayContent []byte) []string {
	matches := QuotedStringRegex.FindAllSubmatch(arrayContent, -1)
	if len(matches) == 0 {
		return nil
	}

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, string(match[1]))
	}
	return result
}
