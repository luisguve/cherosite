package main

import (
	"log"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	app "github.com/luisguve/cherosite/internal/app/cherosite"
	"github.com/luisguve/cherosite/internal/pkg/livedata"
	"github.com/luisguve/cherosite/internal/pkg/router"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc"
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

	// Get session key.
	env, err := godotenv.Read("cookie_hash.env")
	if err != nil {
		log.Fatal(err)
	}
	key, ok := env["SESSION_KEY"]
	if !ok {
		log.Fatal("Missing session key")
	}

	// Create session store.
	store := sessions.NewCookieStore([]byte(key))

	// Create and start hub
	hub := livedata.NewHub()
	go hub.Run(ccc)

	// Get section names mapped to their ids.
	env, err = godotenv.Read("sections.env")
	if err != nil {
		log.Fatal(err)
	}

	// Setup router and routes
	router := router.New(tpl, ccc, store, hub, env)
	router.SetupRoutes()

	// Start app
	a := app.New(router)
	log.Fatal(a.Run())
}
