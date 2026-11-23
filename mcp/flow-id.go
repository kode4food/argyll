package mcp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const flowIDHashLen = 8

// flowIDSeq separates IDs generated within one clock tick
var flowIDSeq atomic.Uint64

// newFlowID joins the first goal set's Step IDs and appends a short hash of the current time
func newFlowID(goals api.Goals) api.FlowID {
	steps := make([]string, 0, len(goals.Steps))
	for _, sid := range goals.Steps {
		steps = append(steps, string(sid))
	}
	var seed [16]byte
	binary.BigEndian.PutUint64(seed[:8], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(seed[8:], flowIDSeq.Add(1))
	sum := sha256.Sum256(seed[:])
	pfx := api.SanitizeID(strings.Join(steps, "-"))
	return api.FlowID(pfx + "-" + hex.EncodeToString(sum[:])[:flowIDHashLen])
}
