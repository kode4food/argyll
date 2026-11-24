package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const problemBodyLimit = 4 * 1024 // 4 KB

func TestReadHTTPProblem(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		detail string
	}{
		{
			name:   "problem JSON",
			body:   `{"title":"Rejected","detail":"invalid token","status":500}`,
			detail: "invalid token",
		},
		{
			name:   "text",
			body:   "  upstream unavailable\n",
			detail: "upstream unavailable",
		},
		{
			name:   "native JSON",
			body:   `{"error":{"message":"bad model"}}`,
			detail: `{"error":{"message":"bad model"}}`,
		},
		{
			name: "empty",
		},
		{
			name:   "bounded",
			body:   strings.Repeat("x", problemBodyLimit+1),
			detail: strings.Repeat("x", problemBodyLimit),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(tc.body)),
				Request: httptest.NewRequest(
					http.MethodGet, "https://user:secret@host/path?key=secret", nil,
				),
			}
			p := api.ReadHTTPProblem(res)
			assert.Equal(t, http.StatusUnauthorized, p.Status)
			assert.Equal(t, tc.detail, p.Detail)
			assert.Equal(t, "https://host/path", p.Instance)
			assert.NotEmpty(t, p.Error())
		})
	}
}

func TestReadHTTPProblemFailure(t *testing.T) {
	p := api.ReadHTTPProblem(&http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(iotest.ErrReader(io.ErrUnexpectedEOF)),
	})
	assert.Equal(t, http.StatusBadGateway, p.Status)
	assert.Contains(t, p.Detail, io.ErrUnexpectedEOF.Error())
}
