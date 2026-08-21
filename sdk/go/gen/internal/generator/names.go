package generator

import (
	"strings"
	"unicode"
)

// SnakeCase converts a Go identifier to an Argyll attribute name
func SnakeCase(s string) string {
	return strings.Join(splitWords(s), "_")
}

// KebabCase converts a Go identifier to an Argyll step ID
func KebabCase(s string) string {
	return strings.Join(splitWords(s), "-")
}

// TitleCase converts a Go identifier to a human readable step name
func TitleCase(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = capitalize(w)
	}
	return strings.Join(words, " ")
}

// ExportedName converts an Argyll attribute name to a Go field name
func ExportedName(s string) string {
	var sb strings.Builder
	for _, w := range splitWords(s) {
		sb.WriteString(capitalize(w))
	}
	return sb.String()
}

func splitWords(s string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = nil
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case r == '_' || r == '-' || unicode.IsSpace(r):
			flush()
		case boundary(rs, i):
			flush()
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

func boundary(rs []rune, i int) bool {
	if i == 0 || !unicode.IsUpper(rs[i]) {
		return false
	}
	prev := rs[i-1]
	if unicode.IsLower(prev) || unicode.IsDigit(prev) {
		return true
	}
	return unicode.IsUpper(prev) && i+1 < len(rs) && unicode.IsLower(rs[i+1])
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}
