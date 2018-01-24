package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

type HetznerContext struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

type HetznerConfig struct {
	ActiveContextName string           `json:"active_context_name"`
	Contexts          []HetznerContext `json:"contexts"`
}

type AppConfig struct {
	Client         *hcloud.Client
	Context        context.Context
	CurrentContext *HetznerContext
	Config         *HetznerConfig
}

var DefaultConfigPath string
var Config HetznerConfig
var AppConf AppConfig = AppConfig{}

func (config HetznerConfig) WriteCurrentConfig() {
	configFileName := filepath.Join(DefaultConfigPath, "config.json")
	configJson, err := json.Marshal(&config)

	if err == nil {
		err = ioutil.WriteFile(configFileName, configJson, 0666)

		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal(err)
	}

	Config = config
}

func (config *HetznerConfig) AddContext(context HetznerContext) {
	config.Contexts = append(config.Contexts, context)
}

func (app *AppConfig) SwitchContextByName(name string) error {
	ctx, err := app.FindContextByName(name)

	if err != nil {
		return err
	}

	app.CurrentContext = ctx
	app.Config.ActiveContextName = ctx.Name

	return nil
}

func (app *AppConfig) FindContextByName(name string) (*HetznerContext, error) {

	for _, ctx := range app.Config.Contexts {
		if ctx.Name == name {

			return &ctx, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("context '%s' not found", name))
}

func init() {
	usr, err := user.Current()
	if err != nil {
		return
	}
	if usr.HomeDir != "" {
		DefaultConfigPath = filepath.Join(usr.HomeDir, ".hetzner-kube")
	}

	AppConf = AppConfig{
		Context: context.Background(),
	}
	makeConfigIfNotExists()
}

func makeConfigIfNotExists() {
	if _, err := os.Stat(DefaultConfigPath); os.IsNotExist(err) {
		os.MkdirAll(DefaultConfigPath, 0755)
	}

	configFileName := filepath.Join(DefaultConfigPath, "config.json")

	if _, err := os.Stat(configFileName); os.IsNotExist(err) {

		// create a empty file with contexts available
		_, err = os.Create(configFileName)
		if err != nil {
			log.Fatal(err)
		}

		AppConf.Config = new(HetznerConfig)
		AppConf.Config.WriteCurrentConfig()

	} else {
		configFileContent, err := ioutil.ReadFile(configFileName)

		if err != nil {
			log.Fatal(err)
		}

		json.Unmarshal(configFileContent, &AppConf.Config)
	}
}
