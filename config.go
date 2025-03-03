package braza

import (
	"crypto/rsa"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v2"
)

/*
Usage:

	env := "dev"
	devConfig := braza.NewConfig()
	prodConfig := braza.NewConfig()
	var cfg  *Config

	if env == "dev"{
		cfg = devConfig
	} else {
		cfg = prodConfig
	}
	app := braza.NewApp(cfg)
	...
*/
func NewConfig() *Config                    { return &Config{} }
func NewConfigFromFile(file string) *Config { c := &Config{}; c.SetupFromFile(file); return c }

type Config struct {
	/*

	 */
	Env            string // environmnt (default 'development')
	SecretKey      string // for sign session (default '')
	Servername     string // for build url routes and route match (default '')
	ListeningInTLS bool   // UrlFor return a URl with schema in "https:" (default 'false')

	TemplateFolder          string // for render Templates Html. Default "templates/"
	TemplateFuncs           template.FuncMap
	DisableParseFormBody    bool // Disable default parse of Request.Form -> if true, use Request.ParseForm()
	DisableTemplateReloader bool // if app in dev mode, disable template's reload (default false)

	StaticFolder  string // for serve static files (default '/assets')
	StaticUrlPath string // url uf request static file (default '/assets')
	DisableStatic bool   // disable static endpoint for serving static files (default false)

	Silent             bool   // don't print logs (default false)
	LogFile            string // save log info in file (default '')
	DotenvFileName     string
	DisableFileWatcher bool // disable autoreload in dev mode (default false)

	SessionExpires          time.Duration // (default 30 minutes)
	SessionPermanentExpires time.Duration // (default 31 days)

	SessionPublicKey  *rsa.PublicKey
	SessionPrivateKey *rsa.PrivateKey

	serverport        string
	defaultWsUpgrader *websocket.Upgrader
}

func (c *Config) checkConfig() {
	if c.Env == "" {
		c.Env = "development"
	}
	if c.TemplateFolder == "" {
		c.TemplateFolder = "templates/"
	}
	if c.TemplateFuncs == nil {
		c.TemplateFuncs = make(template.FuncMap)
	}
	if !c.DisableStatic {
		if c.StaticFolder == "" {
			c.StaticFolder = "assets"
		}
		if c.StaticUrlPath == "" {
			c.StaticUrlPath = "/assets"
		}
	}
	if c.SessionExpires == 0 {
		c.SessionExpires = time.Minute * 30
	}
	if c.SessionPermanentExpires == 0 {
		c.SessionPermanentExpires = time.Hour * 744
	}
	if c.defaultWsUpgrader == nil {
		c.defaultWsUpgrader = &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
	}
}

// setup config from json, yalm
func (c *Config) SetupFromFile(filename string) error {
	f, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	ext := filepath.Ext(filename)
	switch ext {
	case ".json":
		if err := json.Unmarshal(f, c); err != nil {
			return err
		}
	case ".yalm":
		if err := yaml.Unmarshal(f, c); err != nil {
			return err
		}
	}
	c.checkConfig()
	return nil
}
