package main

import (
	"os"

	"github.com/kode4food/argyll/archiver/internal/cmd"
)

type fileConfig struct {
	cmd.Config
	SinkPath string
}

const defaultSinkPath = "/dev/null"

func loadFileConfig() fileConfig {
	cfg := fileConfig{
		SinkPath: defaultSinkPath,
	}
	cmd.LoadConfig(&cfg.Config)

	if sinkPath := os.Getenv("ARCHIVE_SINK_PATH"); sinkPath != "" {
		cfg.SinkPath = sinkPath
	}

	return cfg
}
