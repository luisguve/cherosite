package cherosite

import (
	"log"
	"net/http"
	"time"
)

func (app *App) Run() error {
	log.Println("Running. Open localhost:8000 in the browser.")
	return app.srv.ListenAndServe()
}

func New(h http.Handler, addr, port string) *App {
	return &App{
		srv: &http.Server{
			Handler:      h,
			Addr:         addr + port,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
			IdleTimeout:  time.Second * 60,
		},
	}
}

type App struct {
	srv *http.Server
}
