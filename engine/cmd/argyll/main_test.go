package main_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestMainExitsOnStoreError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/argyll")
	cmd.Env = append(os.Environ(),
		"ENGINE_REDIS_ADDR=127.0.0.1:0",
		"FLOW_REDIS_ADDR=127.0.0.1:0",
	)

	if err := cmd.Run(); err == nil {
		t.Fatalf("expected process to exit with error")
	}

	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("process did not exit within timeout")
	}
}
