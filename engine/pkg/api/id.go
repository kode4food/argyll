package api

import (
	"regexp"
	"strings"
)

type (
	// FlowID is a unique identifier for a flow
	FlowID string

	// StepID is a unique identifier for a step
	StepID string

	// FlowStep identifies a step execution within a flow
	FlowStep struct {
		FlowID FlowID
		StepID StepID
	}
)

// InvalidIDChars matches characters not permitted in flow and step IDs. Valid
// characters are: letters, digits, underscore, dot, hyphen, plus, space
var InvalidIDChars = regexp.MustCompile(`[^a-zA-Z0-9_.\-+ ]`)

// SanitizeID lowercases an ID, removes invalid characters, replaces spaces
// with hyphens, and trims leading and trailing hyphens
func SanitizeID[T ~string](id T) T {
	lower := strings.ToLower(string(id))
	sanitized := InvalidIDChars.ReplaceAllString(lower, "")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	return T(strings.Trim(sanitized, "-"))
}
