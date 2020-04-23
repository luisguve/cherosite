package main

import (
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/luisguve/cheropatilla/internal/app"
)

func main() {
	router := mux.NewRouter()
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	a := app.New(router, store)
	a.SetupRoutes()
	a.Run()
}
