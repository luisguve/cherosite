package main

import (
	"os"
	"log"

	"google.golang.org/grpc"
	"github.com/gorilla/sessions"
	"github.com/luisguve/cheropatilla/internal/app"
	"github.com/luisguve/cheropatilla/internal/pkg/livedata"
	"github.com/luisguve/cheropatilla/internal/pkg/router"
	"github.com/luisguve/cheropatilla/internal/pkg/templates"
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

func main() {
	// Get new template engine
	tpl := templates.New()

	// Establish connection with gRPC server
	conn, err := grpc.Dial(os.Getenv("GRPC_ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create gRPC crud client
	ccc := pb.NewCrudCheropatillaClient(conn)

	// Get session store
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

	// Create and start hub
	hub := livedata.NewHub()
	go hub.Run(ccc)

	// Setup router and routes
	router := router.New(tpl, ccc, store, hub)
	router.SetupRoutes()

	// Start app
	a := app.New(router)
	log.Fatal(a.Run())
}
