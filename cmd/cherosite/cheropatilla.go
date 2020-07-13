package main

import (
	"os"
	"log"

	"google.golang.org/grpc"
	"github.com/gorilla/sessions"
	app "github.com/luisguve/cherosite/internal/app/cherosite"
	"github.com/luisguve/cherosite/internal/pkg/livedata"
	"github.com/luisguve/cherosite/internal/pkg/router"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
)

func main() {
	// Get new template engine
	tpl := templates.Setup()

	const address = "localhost:50051"
	// Establish connection with gRPC server
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create gRPC crud client
	ccc := pbApi.NewCrudCheropatillaClient(conn)

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
