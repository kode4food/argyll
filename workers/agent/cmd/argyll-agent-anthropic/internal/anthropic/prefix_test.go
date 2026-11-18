package anthropic_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-anthropic/internal/anthropic"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

func TestPromptPrefix(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("PREFIX_THRESHOLDS", "")
	for _, tc := range []struct {
		name          string
		compactBytes  int
		expandedBytes int
		expectCompact bool
		overrides     string
	}{
		{
			name:          "compact qualifies",
			compactBytes:  900,
			expandedBytes: 1230,
			expectCompact: true,
		},
		{
			name:          "expanded qualifies",
			compactBytes:  374,
			expandedBytes: 703,
		},
		{
			name:          "override raises threshold",
			compactBytes:  374,
			expandedBytes: 703,
			expectCompact: true,
			overrides:     `{"claude-haiku-5-5":1024}`,
		},
		{
			name:          "compact just above minimum",
			compactBytes:  680,
			expandedBytes: 1010,
		},
		{
			name:          "expanded just below minimum",
			compactBytes:  280,
			expandedBytes: 610,
		},
		{
			name:          "both short",
			compactBytes:  100,
			expandedBytes: 430,
			expectCompact: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.overrides != "" {
				path := filepath.Join(t.TempDir(), "prefix-thresholds.json")
				err := os.WriteFile(path, []byte(tc.overrides), 0o600)
				assert.NoError(t, err)
				t.Setenv("PREFIX_THRESHOLDS", path)
			}
			compact := strings.Repeat("c", tc.compactBytes)
			expanded := strings.Repeat("e", tc.expandedBytes)
			system := expanded
			if tc.expectCompact {
				system = compact
			}
			handler := func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/messages", r.URL.Path)
				var body map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				expected := []any{map[string]any{
					"type":          "text",
					"text":          system,
					"cache_control": map[string]any{"type": "ephemeral"},
				}}
				assert.Equal(t, expected, body["system"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w,
					`{"content":[],"usage":{"input_tokens":600,"output_tokens":1}}`)
			}
			srv := httptest.NewServer(http.HandlerFunc(handler))
			defer srv.Close()
			p, err := anthropic.New(map[string]string{anthropic.Name: srv.URL})
			if !assert.NoError(t, err) {
				return
			}
			c, err := p.Open(&worker.Chat{
				Config: &api.AgentConfig{
					Model:     "claude-haiku-5-5",
					MaxTokens: 5,
				},
				System:        expanded,
				CompactSystem: compact,
				User:          `{"your-name":"Ada"}`,
				Tools: []worker.ToolSpec{
					{
						Name:        "finish",
						Description: "Report the outputs and end the task",
						Schema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"the-greeting": map[string]any{},
							},
							"required":             []any{"the-greeting"},
							"additionalProperties": false,
						},
					},
				},
			})
			if !assert.NoError(t, err) {
				return
			}
			_, err = c.Turn(t.Context())
			assert.NoError(t, err)
		})
	}
}
