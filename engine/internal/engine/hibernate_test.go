package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMemoryInfo(t *testing.T) {
	tests := []struct {
		name     string
		info     string
		wantUsed int64
		wantMax  int64
	}{
		{
			name: "valid memory info with limit",
			info: `# Memory
used_memory:1073741824
maxmemory:2147483648
used_memory_human:1.00G
maxmemory_human:2.00G`,
			wantUsed: 1073741824,
			wantMax:  2147483648,
		},
		{
			name: "maxmemory set to zero",
			info: `# Memory
used_memory:1649344
used_memory_human:1.57M
maxmemory:0
maxmemory_human:0B
maxmemory_policy:noeviction`,
			wantUsed: 1649344,
			wantMax:  0,
		},
		{
			name: "no maxmemory line",
			info: `# Memory
used_memory:1073741824
used_memory_human:1.00G`,
			wantUsed: 1073741824,
			wantMax:  0,
		},
		{
			name:     "empty info",
			info:     "",
			wantUsed: 0,
			wantMax:  0,
		},
		{
			name: "with extra whitespace",
			info: `  used_memory:1073741824
  maxmemory:2147483648  `,
			wantUsed: 1073741824,
			wantMax:  2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUsed, gotMax := parseMemoryInfo(tt.info)
			assert.Equal(t, tt.wantUsed, gotUsed)
			assert.Equal(t, tt.wantMax, gotMax)
		})
	}
}
