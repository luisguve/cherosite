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
	// Get config variables.
	config, err := godotenv.Read("C:/cherosite_files/config.env")
	if err != nil {
		log.Fatal(err)
	}
	environment, ok := config["ENV"]
	if !ok {
		log.Fatal("Missing ENV in config.env")
	}
	// Get address and port
	addr, ok := config["BIND_ADDR"]
	if !ok {
		log.Fatal("Missing BIND_ADDR in config.env")
	}
	port, ok := config["PORT"]
	if !ok {
		log.Fatal("Misssing PORT in config.env")
	}
	// Get new template engine
	tpl := templates.Setup(environment, port)

	// Get grpc config variables.
	grpcConfig, err := godotenv.Read("C:/cheroshared_files/grpc_config.env")
	if err != nil {
		log.Fatal(err)
	}
	address, ok := grpcConfig["BIND_ADDR"]
	if !ok {
		log.Fatal("Missing BIND_ADDR in grpc_config.env")
	}
	// Establish connection with gRPC server
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create gRPC crud client
	ccc := pbApi.NewCrudCheropatillaClient(conn)

	// Get session key.
	env, err := godotenv.Read("C:/cherosite_files/cookie_hash.env")
	if err != nil {
		log.Fatal(err)
	}
	key, ok := env["SESSION_KEY"]
	if !ok {
		log.Fatal("Missing SESSION_KEY in cookie_hash.env")
	}

	// Create session store.
	store := sessions.NewFilesystemStore("tmp/sess", []byte(key))
	// Set no limit on length of sessions.
	store.MaxLength(0)
	// Create and start hub
	hub := livedata.NewHub()
	go hub.Run(ccc)

	// Get section names mapped to their ids.
	sections, err := godotenv.Read("C:/cheroshared_files/sections.env")
	if err != nil {
		log.Fatal(err)
	}

	// Setup router and routes
	router := router.New(tpl, ccc, store, hub, sections)
	router.SetupRoutes()

	// Start app
	a := app.New(router, addr, port)
	log.Fatal(a.Run())
}
