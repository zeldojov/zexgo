package app

import (
	"errors"

	viewspkg "github.com/zeldojov/zexgo/internal/views"
)

var (
	ErrTemplateRender = errors.New("template render error")
	ErrTemplateWrite  = errors.New("template write error")
)

type App struct {
	views  *viewspkg.Views
	Config Config
	State  State
}

type Config struct {
	Environment   string
	TemplatesPath string
	StaticPath    string
}

type State struct {
	// aplikaciono stanje
}

// region helpers

// endregion helpers
// region API

func NewApp(config Config) *App {
	return &App{
		Config: config,
	}
}

// endregion API
