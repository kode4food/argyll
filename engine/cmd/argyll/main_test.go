package main_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMainExitsOnStoreError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/argyll")
	cmd.Env = append(os.Environ(),
		"ENGINE_REDIS_ADDR=127.0.0.1:0",
		"FLOW_REDIS_ADDR=127.0.0.1:0",
	)

	err := cmd.Run()
	assert.Error(t, err)
	assert.NotEqual(t, context.DeadlineExceeded, ctx.Err())
}
