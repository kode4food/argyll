package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const fetchTimeout = 20 * time.Second

var (
	ErrHTTPStatus   = errors.New("could not fetch model documentation")
	ErrInvalidRate  = errors.New("invalid model rate")
	ErrInvalidTable = errors.New("invalid model defaults table")
)

// Fetch reads a provider’s published documentation
func Fetch(url string) (string, error) {
	client := &http.Client{Timeout: fetchTimeout}
	res, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s: %s", ErrHTTPStatus, url, res.Status)
	}
	body, err := io.ReadAll(res.Body)
	return string(body), err
}

// Cells returns the trimmed cells of a Markdown table row
func Cells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	cells := strings.Split(strings.Trim(line, "|"), "|")
	for idx, cell := range cells {
		cells[idx] = strings.TrimSpace(cell)
	}
	return cells
}

// Rate parses a per-million-token price or an unavailable value
func Rate(cell string) (*float64, error) {
	if cell == "-" || cell == "—" {
		return nil, nil
	}
	if cell == "Free" {
		return new(float64), nil
	}
	amount, ok := strings.CutPrefix(cell, "$")
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRate, cell)
	}
	num, err := strconv.ParseFloat(strings.ReplaceAll(amount, ",", ""), 64)
	if err != nil || num < 0 || math.IsNaN(num) || math.IsInf(num, 0) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRate, cell)
	}
	return &num, nil
}

// ReadTable loads existing defaults before a refresh
func ReadTable[T any](path string) (map[string]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var table map[string]T
	if err := json.Unmarshal(data, &table); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidTable, path, err)
	}
	if table == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTable, path)
	}
	return table, nil
}

// WriteTable replaces a defaults file atomically when its values change
func WriteTable[T any](path string, table map[string]T) error {
	before, err := ReadTable[T](path)
	if err != nil {
		return err
	}
	old, err := json.Marshal(before)
	if err != nil {
		return err
	}
	data, err := json.Marshal(table)
	if err != nil || bytes.Equal(old, data) {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".model-defaults-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := errors.Join(enc.Encode(table), tmp.Close()); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
