package main

import "os"

type fileConfig struct {
	SinkPath string
}

const defaultSinkPath = "/dev/null"

func loadFileConfig() fileConfig {
	cfg := fileConfig{
		SinkPath: defaultSinkPath,
	}

	if sinkPath := os.Getenv("ARCHIVE_SINK_PATH"); sinkPath != "" {
		cfg.SinkPath = sinkPath
	}

	return cfg
}
