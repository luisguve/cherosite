package app

import (
	"net/http"
	"time"

	"github.com/luisguve/cheropatilla/internal/pkg/pagination"
)

func (app *App) Run() error {
	return app.srv.ListenAndServe()
}

func New(h http.Handler) *App {
	// Register pagination
	pagination.RegisterSessionTypes()
	return &App{
		srv: &http.Server{
			Handler: h,	
			Addr:    "127.0.0.1:8000",
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
        	IdleTimeout:  time.Second * 60,
		},
	}
}

type App struct {
	srv *http.Server
}
