package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kode4food/argyll/engine/cmd/argyll-agent-check/internal/check"
)

const (
	exitFailed = 1
	exitUsage  = 2

	defaultWorkerURL = "http://localhost:8070"
)

var (
	ErrModelRequired = errors.New("-provider and -model are required")
)

func main() {
	cfg := &check.Config{}
	flag.StringVar(&cfg.WorkerURL, "worker", defaultWorkerURL, "agent worker base URL")
	flag.StringVar(&cfg.Provider, "provider", "", "provider the live attempt runs on")
	flag.StringVar(&cfg.Model, "model", "", "model the live attempt runs")
	flag.StringVar(&cfg.Listen, "listen", "localhost:0", "address receiving the attempt's outcome")
	flag.StringVar(&cfg.CallbackURL, "callback", "",
		"base URL the worker reaches -listen at, when it differs")
	flag.DurationVar(&cfg.Timeout, "timeout", time.Minute, "how long the live attempt may run")
	flag.Parse()

	if cfg.Provider == "" || cfg.Model == "" {
		_, _ = fmt.Fprintln(os.Stderr, ErrModelRequired)
		os.Exit(exitUsage)
	}
	results, err := check.Run(cfg)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailed)
	}
	if !report(results) {
		os.Exit(exitFailed)
	}
}

func report(results []check.Result) bool {
	passed := true
	for _, r := range results {
		if r.Err != nil {
			passed = false
			fmt.Printf("FAIL %s: %v\n", r.Name, r.Err)
			continue
		}
		fmt.Printf("PASS %s\n", r.Name)
	}
	return passed
}
