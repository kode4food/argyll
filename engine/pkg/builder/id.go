package builder

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

type (
	// FlowID is a unique identifier for a flow
	FlowID string

	// StepID is a unique identifier for a step
	StepID string
)

// NewFlowID generates a unique flow ID with a readable prefix
func NewFlowID(prefix string) FlowID {
	prefix = strings.ToLower(prefix)
	prefix = strings.ReplaceAll(prefix, " ", "-")
	suffix := randomHex(6)
	return FlowID(prefix + "-" + suffix)
}

func randomHex(length int) string {
	bytes := make([]byte, (length+1)/2)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
