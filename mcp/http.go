package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const contentTypeJSON = "application/json"

func (s *Server) httpGet(path string) (any, error) {
	req, err := http.NewRequest(http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.New(string(body))
	}
	return decodeJSON(body)
}

func (s *Server) httpPost(path string, payload any) (any, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(
		http.MethodPost, s.baseURL+path, bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.New(string(body))
	}
	return decodeJSON(body)
}

func decodeJSON(body []byte) (any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, errors.New("empty response body")
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
