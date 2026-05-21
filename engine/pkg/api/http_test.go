package api_test

import (
	"net/http"
	"testing"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNewProblem(t *testing.T) {
	p := api.NewProblem(404, "flow not found")
	as := assert.New(t)
	as.Equal("about:blank", p.Type)
	as.Equal("Not Found", p.Title)
	as.Equal(404, p.Status)
	as.Equal("flow not found", p.Detail)
}

func TestProblemDetailsError(t *testing.T) {
	as := assert.New(t)

	p := &api.ProblemDetails{
		Detail: "something went wrong",
		Title:  "Error",
		Status: 500,
	}
	as.Equal("something went wrong", p.Error())

	p = &api.ProblemDetails{Title: "Bad Gateway", Status: 502}
	as.Equal("Bad Gateway", p.Error())

	p = &api.ProblemDetails{Status: 503}
	as.Equal("Service Unavailable", p.Error())

	var nilProblem *api.ProblemDetails
	as.Equal("", nilProblem.Error())
}

func TestSetMetadataHeaders(t *testing.T) {
	as := assert.New(t)

	header := make(http.Header)
	meta := api.Metadata{
		api.MetaFlowID:       "flow-123",
		api.MetaStepID:       "step-456",
		api.MetaReceiptToken: "token-789",
		api.MetaWebhookURL:   "https://example.com/hook",
	}

	api.SetMetadataHeaders(header, meta)

	as.Equal("flow-123", header.Get(api.HeaderFlowID))
	as.Equal("step-456", header.Get(api.HeaderStepID))
	as.Equal("token-789", header.Get(api.HeaderReceiptToken))
	as.Equal("https://example.com/hook", header.Get(api.HeaderWebhookURL))
}

func TestSetMetadataHeadersPartial(t *testing.T) {
	as := assert.New(t)

	header := make(http.Header)
	meta := api.Metadata{
		api.MetaFlowID: "flow-123",
	}

	api.SetMetadataHeaders(header, meta)

	as.Equal("flow-123", header.Get(api.HeaderFlowID))
	as.Empty(header.Get(api.HeaderStepID))
	as.Empty(header.Get(api.HeaderReceiptToken))
	as.Empty(header.Get(api.HeaderWebhookURL))
}

func TestMetadataFromHeaders(t *testing.T) {
	as := assert.New(t)

	header := make(http.Header)
	header.Set(api.HeaderFlowID, "flow-123")
	header.Set(api.HeaderStepID, "step-456")
	header.Set(api.HeaderReceiptToken, "token-789")
	header.Set(api.HeaderWebhookURL, "https://example.com/hook")

	meta := api.MetadataFromHeaders(header)

	as.Equal("flow-123", meta[api.MetaFlowID])
	as.Equal("step-456", meta[api.MetaStepID])
	as.Equal("token-789", meta[api.MetaReceiptToken])
	as.Equal("https://example.com/hook", meta[api.MetaWebhookURL])
}

func TestMetadataFromHeadersEmpty(t *testing.T) {
	as := assert.New(t)

	header := make(http.Header)
	meta := api.MetadataFromHeaders(header)

	as.Empty(meta)
}

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
