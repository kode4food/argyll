package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/log"
)

func TestNewUsesInfoLevel(t *testing.T) {
	logger := log.New("svc", "dev", "1.0.0")
	ctx := context.Background()

	assert.False(t, logger.Handler().Enabled(ctx, slog.LevelDebug))
	assert.True(t, logger.Handler().Enabled(ctx, slog.LevelInfo))
}

func TestNewWithLevelOutputsBaseAttrs(t *testing.T) {
	output := captureStdout(t, func() {
		logger := log.NewWithLevel("svc-name", "prod", "2.3.4", slog.LevelDebug)
		logger.Info("hello", slog.Int("count", 1))
	})

	var got map[string]any
	assert.NoError(t, json.Unmarshal(output, &got))

	assertAttr(t, got, "service", "svc-name")
	assertAttr(t, got, "env", "prod")
	assertAttr(t, got, "version", "2.3.4")
	assertAttr(t, got, "count", float64(1))
}

func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
	_ = r.Close()
	return bytes.TrimSpace(buf.Bytes())
}

func assertAttr(t *testing.T, got map[string]any, key string, expected any) {
	t.Helper()
	val, ok := got[key]
	assert.True(t, ok)
	assert.Equal(t, expected, val)
}
