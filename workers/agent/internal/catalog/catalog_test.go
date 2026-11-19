package catalog_test

import (
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
	_, err = catalog.Fetch(":")
	assert.Error(t, err)
}

func TestRate(t *testing.T) {
	for cell, want := range map[string]*float64{
		"$0.125": new(float64(0.125)),
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

func TestInvalidTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thresholds.json")
	_, err := catalog.ReadTable[int](path)
	assert.ErrorIs(t, err, os.ErrNotExist)
	for _, data := range []string{"null", "{"} {
		assert.NoError(t, os.WriteFile(path, []byte(data), 0o600))
		err := catalog.WriteTable(path, map[string]int{"model": 1024})
		assert.ErrorIs(t, err, catalog.ErrInvalidTable)
		after, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, data, string(after))
	}
}
