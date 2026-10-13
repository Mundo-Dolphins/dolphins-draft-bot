package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/outbound/espn"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/outbound/storage"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/outbound/telegram"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/config"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/logging"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tools <validate-config|print-state|send-test-message|parse-fixture|simulate-payload>")
		os.Exit(1)
	}
	cmd := os.Args[1]
	configPath := "config/config.yaml"
	envPath := ".env"
	requireTelegram := cmd != "print-state" && cmd != "parse-fixture" && cmd != "simulate-payload"
	cfg, err := config.LoadWithOptions(configPath, envPath, config.LoadOptions{RequireTelegram: requireTelegram})
	if err != nil && cmd != "validate-config" {
		fmt.Println("load config error:", err)
		os.Exit(1)
	}
	logger := logging.New("info", "json")
	switch cmd {
	case "validate-config":
		fs := flag.NewFlagSet("validate-config", flag.ExitOnError)
		fs.StringVar(&configPath, "config", configPath, "config path")
		fs.StringVar(&envPath, "env-file", envPath, "env path")
		_ = fs.Parse(os.Args[2:])
		_, err := config.Load(configPath, envPath)
		if err != nil {
			fmt.Println("invalid config:", err)
			os.Exit(1)
		}
		fmt.Println("config valid")
	case "print-state":
		repo := storage.NewJSONEventRepository(cfg.Storage.Path)
		events, err := repo.List(context.Background())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, e := range events {
			fmt.Printf("%s %s %d %s\n", e.Key, e.Type, e.Overall, e.CreatedAt.Format(time.RFC3339))
		}
	case "send-test-message":
		pub := telegram.NewPublisher(cfg.Telegram.BotToken, cfg.Telegram.ChatID, &http.Client{Timeout: cfg.HTTP.Timeout}, cfg.HTTP.Retries, cfg.HTTP.Backoff)
		if err := pub.Publish(context.Background(), "Mensaje de prueba draft-bot"); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("ok")
	case "parse-fixture":
		if len(os.Args) < 3 {
			fmt.Println("usage: tools parse-fixture <path>")
			os.Exit(1)
		}
		raw, err := os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		parser := espn.NewParser()
		snap, kind, err := parser.Parse(raw)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("kind=%s picks=%d round=%d\n", kind, len(snap.Picks), snap.Status.CurrentRound)
	case "simulate-payload":
		if len(os.Args) < 3 {
			fmt.Println("usage: tools simulate-payload <fixture>")
			os.Exit(1)
		}
		raw, err := os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		parser := espn.NewParser()
		snap, _, err := parser.Parse(raw)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		logger.Info("simulation", "picks", len(snap.Picks), "round", snap.Status.CurrentRound)
		_, _ = io.WriteString(os.Stdout, "simulation completed\n")
	default:
		fmt.Println("unknown command")
		os.Exit(1)
	}
}
