package generator

import (
	"errors"
	"fmt"
	"strings"
)

type (
	// Options are the key:value settings of an argyll tag or directive
	Options []Option

	// Option is a single key:value setting
	Option struct {
		Key   string
		Value string
	}

	// Head is the leading value of a tag or directive, the name it declares and
	// the `(in) -> (out)` attribute spec only a wrap accepts
	Head struct {
		Name  string
		Attrs string
	}
)

const (
	optionSeparator = ";"
	optionAssign    = ":"
	arrow           = "->"
)

var (
	ErrBadOption = errors.New("invalid argyll option")
)

// SplitHead divides the leading value into the name and attribute spec its
// caller allows
func SplitHead(text string) Head {
	cut := len(text)
	if i := strings.Index(text, "("); i >= 0 {
		cut = i
	}
	if i := strings.Index(text, arrow); i >= 0 && i < cut {
		cut = i
	}
	return Head{
		Name:  strings.TrimSpace(text[:cut]),
		Attrs: strings.TrimSpace(text[cut:]),
	}
}

// ParseOptions splits a tag or a directive into the leading value naming the
// thing and the semicolon separated key:value options configuring it
func ParseOptions(text string) (string, Options, error) {
	head, rest, _ := strings.Cut(text, optionSeparator)
	// a leading segment that is itself key:value is an option, not a head
	if strings.Contains(head, optionAssign) {
		head = ""
		rest = text
	}
	var opts Options
	for o := range strings.SplitSeq(rest, optionSeparator) {
		if strings.TrimSpace(o) == "" {
			continue
		}
		key, value, ok := strings.Cut(o, optionAssign)
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" || value == "" {
			return "", nil, fmt.Errorf("%w: %q is not key%svalue",
				ErrBadOption, strings.TrimSpace(o), optionAssign)
		}
		opts = append(opts, Option{Key: key, Value: value})
	}
	return strings.TrimSpace(head), opts, nil
}
