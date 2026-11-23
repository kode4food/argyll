package catalog_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/workers/agent/internal/catalog"
)

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pricing" {
			_, _ = fmt.Fprint(w, "# Pricing")
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	doc, err := catalog.Fetch(srv.URL + "/pricing")
	assert.NoError(t, err)
	assert.Equal(t, "# Pricing", doc)
	_, err = catalog.Fetch(srv.URL + "/missing")
	assert.ErrorIs(t, err, catalog.ErrHTTPStatus)
	status, ok := errors.AsType[*catalog.HTTPError](err)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusNotFound, status.Status)
	}
	assert.ErrorContains(t, err, "404")
	_, err = catalog.Fetch(":")
	assert.Error(t, err)
}

func TestInvalidOutputs(t *testing.T) {
	dir := t.TempDir()
	_, err := catalog.ReadTable[int](dir)
	assert.Error(t, err)
	for _, values := range []any{nil, []int{1}, map[string]any{"bad": make(chan int)}} {
		assert.Error(t, catalog.WriteTables(catalog.Table{
			Path:   filepath.Join(dir, "output.json"),
			Values: values,
		}))
	}
	assert.Error(t, catalog.WriteTables(catalog.Table{
		Path:   filepath.Join(dir, "missing", "output.json"),
		Values: map[string]int{"value": 1},
	}))
}

func TestRate(t *testing.T) {
	for cell, want := range map[string]*float64{
		"$0.125": new(0.125),
		"$1,000": new(float64(1000)),
		"Free":   new(float64),
		"-":      nil,
	} {
		rate, err := catalog.Rate(cell)
		assert.NoError(t, err)
		assert.Equal(t, want, rate)
	}
	for _, cell := range []string{"1.25", "$NaN", "$-1"} {
		_, err := catalog.Rate(cell)
		assert.ErrorIs(t, err, catalog.ErrInvalidRate)
	}
	assert.Equal(t, []string{"Model", "Output"}, catalog.Cells("| Model | Output |"))
	assert.Nil(t, catalog.Cells("Heading"))
}

func TestWriteTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thresholds.json")
	data := []byte(`{"model":512}`)
	assert.NoError(t, os.WriteFile(path, data, 0o600))
	assert.NoError(t, catalog.WriteTable(path, map[string]int{"model": 512}))
	after, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, data, after)

	assert.NoError(t, catalog.WriteTable(path, map[string]int{"model": 1024}))
	table, err := catalog.ReadTable[int](path)
	assert.NoError(t, err)
	assert.Equal(t, 1024, table["model"])
	info, err := os.Stat(path)
	if assert.NoError(t, err) {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestExactNumbers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numbers.json")
	assert.NoError(t, catalog.WriteTable(path, map[string]int64{"model": 9007199254740992}))
	assert.NoError(t, catalog.WriteTable(path, map[string]int64{"model": 9007199254740993}))
	table, err := catalog.ReadTable[int64](path)
	assert.NoError(t, err)
	assert.Equal(t, int64(9007199254740993), table["model"])
}

func TestNewTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thresholds.json")
	table, err := catalog.ReadTable[int](path)
	assert.NoError(t, err)
	assert.Empty(t, table)

	assert.NoError(t, catalog.WriteTable(path, map[string]int{"model": 512}))
	table, err = catalog.ReadTable[int](path)
	assert.NoError(t, err)
	assert.Equal(t, map[string]int{"model": 512}, table)
	info, err := os.Stat(path)
	if assert.NoError(t, err) {
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	}
}

func TestMerge(t *testing.T) {
	existing := map[string]int{
		"kept":    1,
		"updated": 2,
	}
	merged := catalog.Merge(catalog.MergeArgs[int]{
		Existing: existing,
		Published: map[string]int{
			"updated": 3,
			"added":   4,
		},
	})
	assert.Equal(t, map[string]int{
		"kept":    1,
		"updated": 3,
		"added":   4,
	}, merged)
	assert.Equal(t, 2, existing["updated"])
}

func TestInvalidTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thresholds.json")
	for _, data := range []string{"null", "{"} {
		assert.NoError(t, os.WriteFile(path, []byte(data), 0o600))
		err := catalog.WriteTable(path, map[string]int{"model": 1024})
		assert.ErrorIs(t, err, catalog.ErrInvalidTable)
		after, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, data, string(after))
	}
}

func TestWriteTables(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.json")
	second := filepath.Join(dir, "b.json")
	before := []byte(`{"kept":1}`)
	assert.NoError(t, os.WriteFile(first, before, 0o600))
	assert.NoError(t, os.WriteFile(second, []byte(`null`), 0o600))
	err := catalog.WriteTables(
		catalog.Table{
			Path:   first,
			Values: map[string]int{"changed": 2},
		},
		catalog.Table{
			Path:   second,
			Values: map[string]int{"changed": 3},
		},
	)
	assert.ErrorIs(t, err, catalog.ErrInvalidTable)
	after, err := os.ReadFile(first)
	assert.NoError(t, err)
	assert.Equal(t, before, after)
	files, err := filepath.Glob(filepath.Join(dir, ".model-defaults-*"))
	assert.NoError(t, err)
	assert.Empty(t, files)

	assert.NoError(t, os.Remove(second))
	assert.NoError(t, catalog.WriteTables(
		catalog.Table{
			Path:   first,
			Values: map[string]int{"changed": 2},
		},
		catalog.Table{
			Path:   second,
			Values: map[string]int{},
		},
	))
	values, err := catalog.ReadTable[int](first)
	assert.NoError(t, err)
	assert.Equal(t, map[string]int{"changed": 2}, values)
	after, err = os.ReadFile(second)
	assert.NoError(t, err)
	assert.JSONEq(t, `{}`, string(after))
}
