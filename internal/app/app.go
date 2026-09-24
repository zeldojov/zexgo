package app

import (
	"io/fs"

	viewspkg "github.com/zeldojov/zexgo/internal/views"
)

type App struct {
	views  *viewspkg.Views
	static fs.FS
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
