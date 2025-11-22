package builder

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/kode4food/spuds/engine/pkg/api"
)

// NewFlowID generates a unique flow ID with a readable prefix
func NewFlowID(prefix string) api.FlowID {
	prefix = strings.ToLower(prefix)
	prefix = strings.ReplaceAll(prefix, " ", "-")
	suffix := randomHex(6)
	return api.FlowID(prefix + "-" + suffix)
}

func randomHex(length int) string {
	bytes := make([]byte, (length+1)/2)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
