package cherosite

import (
	"log"
	"net/http"
	"time"
)

func (a *App) Run() error {
	log.Printf("Running. Open %s in the browser.\n", a.srv.Addr)
	return a.srv.ListenAndServe()
}

func New(h http.Handler, bindAddr string) *App {
	return &App{
		srv: &http.Server{
			Handler:      h,
			Addr:         bindAddr,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
			IdleTimeout:  time.Second * 60,
		},
	}
}

type App struct {
	srv *http.Server
}
