package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/gorilla/sessions"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbUsers "github.com/luisguve/cheroproto-go/userapi"
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

type sectionConfig struct {
	BindAddress string `toml:"bind_address"`
	Id          string `toml:"id"`
	Name        string `toml:"name"`
}

type sessConfig struct {
	Dir string `toml:"sess_dir"`
	Key string `toml:"sess_secret_key"`
}

type cherositeConfig struct {
	UploadDir      string                `toml:"upload_dir"`
	StaticDir      string                `toml:"static_dir"`
	InternalTplDir string                `toml:"internal_tpl_dir"`
	PublicTplDir   string                `toml:"public_tpl_dir"`
	ServicesConf   map[string]grpcConfig `toml:"services"`
	Sections       []sectionConfig       `toml:"sections"`
	HttpConf       httpConfig            `toml:"http_config"`
	Patillavatars  []string              `toml:"patillavatars"`
	SessEnv        sessConfig            `toml:"session_variables"`
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Absolute path of .toml config file.")

	flag.Parse()

	if configFile == "" {
		log.Fatal("Absolute path of .toml config file must be set.")
	}

	config := cherositeConfig{}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatal(err)
	}

	if err := config.preventDefault(); err != nil {
		log.Fatal(err)
	}

	// Create session store.
	sessDir := config.SessEnv.Dir
	sessKey := []byte(config.SessEnv.Key)
	store := sessions.NewFilesystemStore(sessDir, sessKey)
	// Set no limit on length of sessions.
	store.MaxLength(0)

	// Establish connection with users gRPC service.
	addr := config.ServicesConf["users"].BindAddress
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	usersClient := pbUsers.NewCrudUsersClient(conn)

	// Create and start hub
	hub := livedata.NewHub(usersClient)
	go hub.Run()

	// Establish connection with general gRPC service.
	addr = config.ServicesConf["general"].BindAddress
	conn, err = grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	generalClient := pbApi.NewCrudGeneralClient(conn)

	// Establish connection with section gRPC services.
	var sections []router.Section

	for _, s := range config.Sections {
		conn, err = grpc.Dial(s.BindAddress, grpc.WithInsecure())
		if err != nil {
			log.Fatal("Could not setup dial:", err)
		}
		defer conn.Close()

		sectionClient := pbApi.NewCrudCheropatillaClient(conn)
		section := router.Section{
			Client: sectionClient,
			Id:     s.Id,
			Name:   s.Name,
		}
		sections = append(sections, section)
	}

	// Setup a new templates engine.
	tpl := templates.Setup(config.HttpConf.Env, ":" + config.HttpConf.Port, config.InternalTplDir, config.PublicTplDir)

	// Setup router and routes.
	router := router.New(tpl, usersClient, generalClient, sections, store, hub, config.Patillavatars)
	router.SetupRoutes(config.UploadDir, config.StaticDir)

	// Start app.
	addr = config.HttpConf.BindAddress + ":" + config.HttpConf.Port
	a := app.New(router, addr)
	log.Fatal(a.Run())
}

func (c cherositeConfig) preventDefault() error {
	if c.UploadDir == "" {
		return fmt.Errorf("Missing upload dir.")
	}
	if c.StaticDir == "" {
		return fmt.Errorf("Missing static dir.")
	}
	if c.InternalTplDir == "" {
		return fmt.Errorf("Missing internal tpl dir.")
	}
	if c.PublicTplDir == "" {
		return fmt.Errorf("Missing public tpl dir.")
	}
	if c.ServicesConf == nil {
		return fmt.Errorf("Missing services config.")
	}
	usersSrvConf, ok := c.ServicesConf["users"]
	if !ok {
		return fmt.Errorf("Missing users service config.")
	}
	if err := usersSrvConf.preventDefault("users"); err != nil {
		return err
	}
	generalSrvConf, ok := c.ServicesConf["general"]
	if !ok {
		return fmt.Errorf("Missing general service config.")
	}
	if err := generalSrvConf.preventDefault("general"); err != nil {
		return err
	}
	if len(c.Sections) == 0 {
		return fmt.Errorf("Missing sections config.")
	}
	for _, s := range c.Sections {
		if err := s.preventDefault(); err != nil {
			return err
		}
	}
	if err := c.HttpConf.preventDefault(); err != nil {
		return err
	}
	if len(c.Patillavatars) == 0 {
		return fmt.Errorf("Missing default patillavatars.")
	}
	return nil
}

func (s sectionConfig) preventDefault() error {
	if s.BindAddress == "" {
		return fmt.Errorf("Missing bind address in one or more sections.")
	}
	if s.Id == "" {
		return fmt.Errorf("Missing id in one or more sections.")
	}
	if s.Name == "" {
		return fmt.Errorf("Missing name in one or more sections.")
	}
	return nil
}

func (h httpConfig) preventDefault() error {
	if h.BindAddress == "" {
		return fmt.Errorf("Missing http bind address.")
	}
	if h.Env == "" {
		return fmt.Errorf("Missing env config.")
	}
	if h.Port == "" {
		return fmt.Errorf("Missing http port.")
	}
	return nil
}

func (g grpcConfig) preventDefault(srvName string) error {
	if g.BindAddress == "" {
		return fmt.Errorf("Missing %s service bind address.", srvName)
	}
	return nil
}

func (s sessConfig) preventDefault() error {
	if s.Dir == "" {
		return fmt.Errorf("Missing session dir.")
	}
	if s.Key == "" {
		return fmt.Errorf("Missing session secret key.")
	}
	return nil
}
