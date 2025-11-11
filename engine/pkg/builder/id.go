package builder

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/kode4food/timebox"
)

// NewFlowID generates a unique flow ID with a readable prefix.
// The generated ID is in the format: "prefix-XXXXXX" where XXXXXX is a
// random 6-character hex string ensuring uniqueness.
func NewFlowID(prefix string) timebox.ID {
	prefix = strings.ToLower(prefix)
	prefix = strings.ReplaceAll(prefix, " ", "-")
	suffix := randomHex(6)
	return timebox.ID(prefix + "-" + suffix)
}

func randomHex(length int) string {
	bytes := make([]byte, (length+1)/2)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
