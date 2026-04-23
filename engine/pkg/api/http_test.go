package api_test

import (
	"testing"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestIsProblemJSON(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "exact",
			contentType: api.ProblemJSONContentType,
			expected:    true,
		},
		{
			name:        "with_charset",
			contentType: api.ProblemJSONContentType + "; charset=utf-8",
			expected:    true,
		},
		{
			name:        "plain_json",
			contentType: api.JSONContentType,
			expected:    false,
		},
		{
			name:        "invalid_prefix",
			contentType: api.ProblemJSONContentType + "x",
			expected:    false,
		},
		{
			name:        "malformed",
			contentType: `application/problem+json; charset="`,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := assert.New(t)
			as.Equal(tt.expected, api.IsProblemJSON(tt.contentType))
		})
	}
}
