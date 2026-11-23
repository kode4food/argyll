package mcp_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp"
)

func TestStartFlowGeneratesID(t *testing.T) {
	var ids []string
	hc := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			var body struct {
				ID string `json:"id"`
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			ids = append(ids, body.ID)
			return jsonResponse(http.StatusCreated, []byte(`{"flow_id":"`+body.ID+`"}`)), nil
		}),
	}
	c := newClient(t, mcp.NewServer("http://example", hc))
	goals := map[string]any{
		"steps": []string{"Notification Sender", "audit"},
		"else":  map[string]any{"steps": []string{"fallback"}},
	}

	callToolText(t, c, "start_flow", map[string]any{"goals": goals})
	callToolText(t, c, "start_flow", map[string]any{"goals": goals, "id": ""})
	callToolText(t, c, "start_flow", map[string]any{"goals": goals, "id": "mine"})

	if assert.Len(t, ids, 3) {
		pattern := regexp.MustCompile(`^notification-sender-audit-[0-9a-f]{8}$`)
		assert.Regexp(t, pattern, ids[0])
		assert.Regexp(t, pattern, ids[1])
		assert.NotEqual(t, ids[0], ids[1])
		assert.Equal(t, "mine", ids[2])
	}
}
