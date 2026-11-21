package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type (
	// Table is one model defaults file to publish after all outputs have been prepared
	Table struct {
		Path   string
		Values any
	}

	// HTTPError identifies a documentation endpoint's non-success response
	HTTPError struct {
		URL    string
		Status int
	}

	preparedTable struct {
		path string
		temp string
	}
)

const (
	fetchTimeout             = 20 * time.Second
	newTablePerm os.FileMode = 0o644
)

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
		return "", errors.Join(ErrHTTPStatus, &HTTPError{
			URL:    url,
			Status: res.StatusCode,
		})
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

// MergeArgs names the defaults a refresh starts from and the values it published
type MergeArgs[T any] struct {
	Existing  map[string]T
	Published map[string]T
}

// Merge returns the existing defaults updated with the published values, reporting each existing
// entry that has no published value and so is retained
func Merge[T any](args MergeArgs[T]) map[string]T {
	for _, key := range slices.Sorted(maps.Keys(args.Existing)) {
		if _, ok := args.Published[key]; !ok {
			fmt.Printf("Retained %s without a published value\n", key)
		}
	}
	res := maps.Clone(args.Existing)
	maps.Copy(res, args.Published)
	return res
}

// ReadTable loads existing defaults before a refresh. A missing file has no defaults yet
func ReadTable[T any](path string) (map[string]T, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]T{}, nil
	}
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
	if _, err := ReadTable[T](path); err != nil {
		return err
	}
	return WriteTables(Table{
		Path:   path,
		Values: table,
	})
}

// WriteTables validates and stages every output before replacing any defaults file
func WriteTables(tables ...Table) error {
	var prepared []preparedTable
	defer func() {
		for _, p := range prepared {
			_ = os.Remove(p.temp)
		}
	}()
	for _, table := range tables {
		path, err := prepareTable(table)
		if err != nil {
			return err
		}
		if path != "" {
			prepared = append(prepared, preparedTable{
				path: table.Path,
				temp: path,
			})
		}
	}
	for _, p := range prepared {
		if err := os.Rename(p.temp, p.path); err != nil {
			return err
		}
	}
	return nil
}

// Error describes the failed fetch
func (h *HTTPError) Error() string {
	return fmt.Sprintf("%s: %d", h.URL, h.Status)
}

// prepareTable stages a changed table, preserving permissions and skipping unchanged contents
func prepareTable(table Table) (string, error) {
	path := table.Path
	before, err := ReadTable[json.RawMessage](path)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(table.Values)
	if err != nil {
		return "", err
	}
	if len(data) == 0 || data[0] != '{' {
		return "", fmt.Errorf("%w: %s", ErrInvalidTable, path)
	}
	old, err := json.Marshal(before)
	if err != nil {
		return "", err
	}
	perm := newTablePerm
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
		if bytes.Equal(old, data) {
			return "", nil
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".model-defaults-*")
	if err != nil {
		return "", err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(tmp.Name())
		}
	}()
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, data, "", "  "); err != nil {
		return "", errors.Join(err, tmp.Close())
	}
	formatted.WriteByte('\n')
	_, err = tmp.Write(formatted.Bytes())
	if err := errors.Join(err, tmp.Close()); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return "", err
	}
	complete = true
	return tmp.Name(), nil
}
