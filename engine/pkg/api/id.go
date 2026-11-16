package api

import (
	"regexp"
	"slices"
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

	// NodeID uniquely identifies a cluster node
	NodeID string

	// Sanitizer returns a sanitized copy of itself
	Sanitizer[T any] interface {
		Sanitized() T
	}
)

var (
	// InvalidIDChars matches characters not permitted in flow and step IDs. Valid characters are:
	// letters, digits, underscore, dot, hyphen, plus, space
	InvalidIDChars = regexp.MustCompile(`[^a-zA-Z0-9_.\-+ ]`)

	_ Sanitizer[*Step]                 = (*Step)(nil)
	_ Sanitizer[*Space]                = (*Space)(nil)
	_ Sanitizer[*Feed]                 = (*Feed)(nil)
	_ Sanitizer[*CreateFlowRequest]    = (*CreateFlowRequest)(nil)
	_ Sanitizer[*ExecutionPlanRequest] = (*ExecutionPlanRequest)(nil)
)

// SanitizeID lowercases an ID, removes invalid characters, replaces spaces with hyphens, and trims
// leading and trailing hyphens
func SanitizeID[T ~string](id T) T {
	lower := strings.ToLower(string(id))
	sanitized := InvalidIDChars.ReplaceAllString(lower, "")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	return T(strings.Trim(sanitized, "-"))
}

func sanitizeIDs[T ~string](ids []T) ([]T, bool) {
	if !slices.ContainsFunc(ids, unsanitized[T]) {
		return ids, false
	}
	res := make([]T, len(ids))
	for i, id := range ids {
		res[i] = SanitizeID(id)
	}
	return res, true
}

func unsanitized[T ~string](id T) bool {
	return SanitizeID(id) != id
}
