package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kode4food/argyll/archiver"
	"github.com/kode4food/argyll/archiver/internal/cmd"
)

func main() {
	cfg, err := archiver.LoadFromEnv()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fileCfg := loadFileConfig()

	sink, err := os.OpenFile(
		fileCfg.SinkPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644,
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = sink.Close() }()

	writer, err := archiver.NewWriter(
		func(ctx context.Context, _ string, data []byte) error {
			if _, err := sink.Write(data); err != nil {
				return err
			}
			_, err := sink.Write([]byte("\n"))
			return err
		},
		"",
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmd.Run(cfg, writer, fileCfg.PollInterval); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
