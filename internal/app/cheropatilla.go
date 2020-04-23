package app

import (
	"net/http"
	"time"

	pb "github.com/luisguve/cheropatilla/internal/pkg/cheropatillapb"
	"github.com/luisguve/cheropatilla/internal/pkg/livedata"
)

func (app *App) Run(crudClient *pb.CrudCheropatillaClient) error {
	go app.hub.Run(crudClient)
	return app.srv.ListenAndServe()
}

func New(h http.Handler, hub *livedata.Hub) *App {
	a := &App{
		srv: &http.Server{
			Handler: h,	
			Addr:    "127.0.0.1:8000",
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
        	IdleTimeout:  time.Second * 60,
		},
		hub: hub,
	}
}

type App struct {
	srv *http.Server
	hub *livedata.Hub
}
