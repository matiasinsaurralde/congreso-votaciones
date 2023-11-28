package main

import (
	"os"
	"time"

	"github.com/matiasinsaurralde/congreso-votaciones/pkg/app"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/config"

	"github.com/rs/zerolog"
)

func newLogger() zerolog.Logger {
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})
	return logger.Level(zerolog.DebugLevel).
		With().Timestamp().
		Logger()
}

func main() {
	logger := newLogger()
	logger.Info().Msg("starting")
	cfg, err := config.Load("config.json")
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading config")
	}
	logger.Debug().Any("SampleData", cfg.SampleData).Msg("SampleData")
	app := app.New(cfg, logger)
	if err := app.Init(); err != nil {
		logger.Fatal().Err(err).Msg("initialization error")
	}
	if err := app.Run(os.Args); err != nil {
		logger.Fatal().Err(err).Msg("error running app")
	}
	logger.Info().Msg("done")
}
