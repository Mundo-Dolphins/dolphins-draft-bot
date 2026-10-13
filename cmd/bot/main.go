package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/inbound/runner"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/config"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/logging"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to YAML config")
	envPath := flag.String("env-file", ".env", "path to .env file")
	flag.Parse()

	cfg, err := config.Load(*configPath, *envPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	if err := runner.Run(cfg, logger); err != nil {
		logger.Error("bot finished with error", "error", err)
		os.Exit(1)
	}
}
