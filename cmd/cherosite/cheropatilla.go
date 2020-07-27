package main

import (
	"errors"
	"fmt"
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

func siteConfig(file string, vars ...string) (map[string]string, error) {
	config, err := godotenv.Read(file)
	if err != nil {
		return nil, err
	}
	var result = make(map[string]string)
	if len(vars) > 0 {
		for _, key := range vars {
			val, ok := config[key]
			if !ok {
				errMsg := fmt.Sprintf("Missing %s in %s.", key, file)
				return nil, errors.New(errMsg)
			}
			result[key] = val
		}
	} else {
		result = config
	}
	return result, nil
}

func main() {
	// Get grpc config variables.
	grpcConfig, err := siteConfig("C:/cheroshared_files/grpc_config.env",
		"BIND_ADDR")
	if err != nil {
		log.Fatal(err)
	}

	// Establish connection with gRPC server
	conn, err := grpc.Dial(grpcConfig["BIND_ADDR"], grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create gRPC crud client
	ccc := pbApi.NewCrudCheropatillaClient(conn)

	// Get session key.
	sess, err := siteConfig("C:/cherosite_files/cookie_hash.env", "SESSION_KEY",
		"SESS_DIR")
	if err != nil {
		log.Fatal(err)
	}

	// Create session store.
	store := sessions.NewFilesystemStore(sess["SESS_DIR"], []byte(sess["SESSION_KEY"]))
	// Set no limit on length of sessions.
	store.MaxLength(0)

	// Create and start hub
	hub := livedata.NewHub()
	go hub.Run(ccc)

	// Get section names mapped to their ids.
	sections, err := siteConfig("C:/cheroshared_files/sections.env")
	if err != nil {
		log.Fatal(err)
	}

	// Get config variables.
	config, err := siteConfig("C:/cherosite_files/config.env", "ENV", "BIND_ADDR",
		"PORT")
	if err != nil {
		log.Fatal(err)
	}

	// Setup a new templates engine.
	tpl := templates.Setup(config["ENV"], config["PORT"])

	// Setup router and routes.
	router := router.New(tpl, ccc, store, hub, sections)
	router.SetupRoutes()

	// Start app.
	a := app.New(router, config["BIND_ADDR"], config["PORT"])
	log.Fatal(a.Run())
}
