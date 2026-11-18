package worker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

func TestServe(t *testing.T) {
	for _, tc := range []struct {
		name string
		host string
		port string
	}{
		{name: "configured port", host: "127.0.0.1", port: "invalid-port"},
		{name: "default port", host: "::invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOST", tc.host)
			t.Setenv("PORT", tc.port)
			err := worker.Serve(&worker.Config{}, nil)
			assert.Error(t, err)
		})
	}
}
