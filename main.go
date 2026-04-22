package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/henrywhitakercommify/http-mock/internal/config"
	"github.com/henrywhitakercommify/http-mock/internal/http"
	"github.com/spf13/pflag"
)

var (
	configFile string
	logLevel   string
)

func main() {
	flags := pflag.NewFlagSet("flags", pflag.ExitOnError)
	flags.StringVarP(&configFile, "config", "c", "http-mock.yaml", "The path to the config file")
	flags.StringVar(&logLevel, "log-level", "info", "The level of logs that are outputted")

	if err := flags.Parse(os.Args); err != nil {
		fmt.Printf("Could not read flags: %v\n", err)
		os.Exit(1)
	}

	conf, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Could not load config: %v\n", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel(logLevel),
	})))

	slog.Debug("loaded config", "config", conf)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv, err := http.New(conf.Endpoints)
	if err != nil {
		fmt.Printf("Could not init http server: %v", err)
		os.Exit(1)
	}

	if err := srv.Run(ctx); err != nil {
		fmt.Printf("Could not run http server: %v", err)
	}
}

func slogLevel(level string) slog.Level {
	switch level {
	case "error":
		return slog.LevelError
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}
