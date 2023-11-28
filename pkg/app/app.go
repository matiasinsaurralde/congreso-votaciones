package app

import (
	"os"
	"path/filepath"

	"github.com/matiasinsaurralde/congreso-votaciones/pkg/config"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/processor"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/store"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// App wraps all the components and embeds the urfave CLI app:
type App struct {
	*cli.App
	logger    zerolog.Logger
	cfg       *config.Config
	store     *store.Store
	processor *processor.Processor
}

func (a *App) Init() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if a.cfg.PDFPath == "" {
		a.cfg.PDFPath = filepath.Join(cwd, defaultPDFPath)
	}
	if a.cfg.ImagePath == "" {
		a.cfg.ImagePath = filepath.Join(cwd, defaultImagePath)
	}
	if a.cfg.JSONPath == "" {
		a.cfg.JSONPath = filepath.Join(cwd, defaultJSONPath)
	}
	if a.cfg.StorePath == "" {
		a.cfg.StorePath = filepath.Join(cwd, defaultStorePath)
	}
	if a.cfg.SamplesPath == "" {
		a.cfg.SamplesPath = filepath.Join(cwd, defaultSamplePath)
	}
	for _, d := range []string{
		a.cfg.PDFPath,
		a.cfg.ImagePath,
		a.cfg.JSONPath,
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Init store:
	a.store = store.New(a.cfg, a.logger)

	if a.store.Init(); err != nil {
		return err
	}

	// Init processor:
	a.processor = processor.New(a.cfg, a.store, a.logger)
	return nil
}

func (a *App) fetch(c *cli.Context) error {
	return nil
}

func (a *App) classify(c *cli.Context) error {
	if err := a.processor.Classify(); err != nil {
		return err
	}
	return nil
}

func (a *App) extract(c *cli.Context) error {
	if err := a.processor.Extract(); err != nil {
		return err
	}
	return nil
}

// New takes a configuration and logger and returns app:
func New(cfg *config.Config, logger zerolog.Logger) *App {
	var app App
	app.logger = logger
	app.cfg = cfg
	app.App = &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "descargar",
				Aliases: []string{"d"},
				Usage:   "Descargar documentos de votación del SILPY",
				Action:  app.fetch,
			},
			{
				Name:    "clasificar",
				Aliases: []string{"c"},
				Usage:   "Clasificar documentos de votación",
				Action:  app.classify,
			},
			{
				Name:    "extraer",
				Aliases: []string{"e"},
				Usage:   "Procesar y extraer datos de los documentos de votación",
				Action:  app.extract,
			},
		},
	}
	return &app
}
