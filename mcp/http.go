package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// authorizationKey holds the Authorization header of the MCP request being served
type authorizationKey struct{}

var (
	ErrHTTPStatus        = errors.New("http request failed")
	ErrEmptyResponseBody = errors.New("empty response body")
)

func (s *Server) httpGet(ctx context.Context, path string) (any, error) {
	return s.httpRequest(ctx, http.MethodGet, path, nil)
}

func (s *Server) httpPost(ctx context.Context, path string, payload any) (any, error) {
	return s.httpRequest(ctx, http.MethodPost, path, payload)
}

func (s *Server) httpPut(ctx context.Context, path string, payload any) (any, error) {
	return s.httpRequest(ctx, http.MethodPut, path, payload)
}

func (s *Server) httpDelete(ctx context.Context, path string) (any, error) {
	return s.httpRequest(ctx, http.MethodDelete, path, nil)
}

// httpRequest forwards the caller's Authorization when present
func (s *Server) httpRequest(
	ctx context.Context, method, path string, payload any,
) (any, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", api.JSONContentType)
	}
	if header, ok := ctx.Value(authorizationKey{}).(string); ok && header != "" {
		req.Header.Set("Authorization", header)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: %s", ErrHTTPStatus, string(data))
	}
	return decodeJSON(data)
}

func decodeJSON(body []byte) (any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, ErrEmptyResponseBody
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
