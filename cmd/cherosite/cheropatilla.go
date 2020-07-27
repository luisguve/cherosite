package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	app "github.com/luisguve/cherosite/internal/app/cherosite"
	"github.com/luisguve/cherosite/internal/pkg/livedata"
	"github.com/luisguve/cherosite/internal/pkg/router"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc"
)

type httpConfig struct {
	BindAddress string `toml:"bind_address"`
	Env         string
	Port        string
}

type grpcConfig struct {
	BindAddress string `toml:"bind_address"`
}

type cherositeConfig struct {
	GrpcConf grpcConfig `toml:"grpc_config"`
	HttpConf httpConfig `toml:"http_config"`
	Sections string `toml:"sections"`
	SessEnv  string `toml:"session_variables"`
}

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
	gopath, ok := os.LookupEnv("GOPATH")
	if !ok || gopath == "" {
		log.Fatal("GOPATH must be set.")
	}

	configDir := filepath.Join(gopath, "src", "github.com", "luisguve",
		"cherosite", "cherosite.toml")

	cheroConfig := new(cherositeConfig)
	if _, err := toml.DecodeFile(configDir, cheroConfig); err != nil {
		log.Fatal(err)
	}

	// Establish connection with gRPC server
	conn, err := grpc.Dial(cheroConfig.GrpcConf.BindAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create gRPC crud client
	ccc := pbApi.NewCrudCheropatillaClient(conn)

	// Get session key.
	sess, err := siteConfig(cheroConfig.SessEnv, "SESSION_KEY", "SESS_DIR")
	if err != nil {
		log.Fatal("siteConfig called with SessEnv: ", err)
	}

	// Create session store.
	store := sessions.NewFilesystemStore(sess["SESS_DIR"], []byte(sess["SESSION_KEY"]))
	// Set no limit on length of sessions.
	store.MaxLength(0)

	// Create and start hub
	hub := livedata.NewHub()
	go hub.Run(ccc)

	// Get section names mapped to their ids.
	sections, err := siteConfig(cheroConfig.Sections)
	if err != nil {
		log.Fatal("siteConfig called with Sections: ", err)
	}

	// Setup a new templates engine.
	tpl := templates.Setup(cheroConfig.HttpConf.Env, ":" + cheroConfig.HttpConf.Port)

	// Setup router and routes.
	router := router.New(tpl, ccc, store, hub, sections)
	router.SetupRoutes()

	// Start app.
	a := app.New(router, cheroConfig.HttpConf.BindAddress, ":" + cheroConfig.HttpConf.Port)
	log.Fatal(a.Run())
}
