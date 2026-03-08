package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/focusd-so/focusd/cmd/serve"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

func main() {

	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", "error", err)
	}

	root := &cli.Command{Name: "focusd", Commands: []*cli.Command{serve.Command}}

	if err := root.Run(context.Background(), os.Args); err != nil {
		slog.Error("failed to run command", "error", err)
		os.Exit(1)
	}
}
