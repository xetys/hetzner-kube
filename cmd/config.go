package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// DefaultConfigPath is the path where the default config is located
var DefaultConfigPath string

// AppConf is the default configuration from the local system.
var AppConf = NewAppConfig()

//AppSSHClient is the SSH client
type AppSSHClient struct {
}

// NewAppConfig creates a new AppConfig struct using the locally saved configuration file. If no local
// configuration file is found a new config will be created.
func NewAppConfig() AppConfig {
	usr, err := user.Current()
	if err != nil {
		return AppConfig{}
	}
	if usr.HomeDir != "" {
		DefaultConfigPath = filepath.Join(usr.HomeDir, ".hetzner-kube")
	}

	appConf := AppConfig{
		Context: context.Background(),
	}

	makeConfigIfNotExists(&appConf)
	appConf.SSHClient = clustermanager.NewSSHCommunicator(appConf.Config.SSHKeys, true)
	return appConf
}

//WriteCurrentConfig write the configuration to file
func (config HetznerConfig) WriteCurrentConfig() {
	configFileName := filepath.Join(DefaultConfigPath, "config.json")
	configJSON, err := json.Marshal(&config)

	if err == nil {
		err = ioutil.WriteFile(configFileName, configJSON, 0666)

		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal(err)
	}
}

//AddContext add context to config
func (config *HetznerConfig) AddContext(context HetznerContext) {
	config.Contexts = append(config.Contexts, context)
}

//AddSSHKey add a new SSH key to config
func (config *HetznerConfig) AddSSHKey(key clustermanager.SSHKey) {
	config.SSHKeys = append(config.SSHKeys, key)
}

//DeleteSSHKey remove the SSH key from config
func (config *HetznerConfig) DeleteSSHKey(name string) error {

	index, err := config.FindSSHKeyByName(name)
	if err != nil {
		return err
	}

	config.SSHKeys = append(config.SSHKeys[:index], config.SSHKeys[index+1:]...)

	return nil
}

//FindSSHKeyByName find a SSH key in config by name
func (config *HetznerConfig) FindSSHKeyByName(name string) (int, error) {
	for i, v := range config.SSHKeys {
		if v.Name == name {
			return i, nil
		}
	}

	return -1, fmt.Errorf("unable to find '%s' SSH key", name)
}

//AddCluster add a cluster in config
func (config *HetznerConfig) AddCluster(cluster clustermanager.Cluster) {
	for i, v := range config.Clusters {
		if v.Name == cluster.Name {
			config.Clusters[i] = cluster
			return
		}
	}

	config.Clusters = append(config.Clusters, cluster)
}

//DeleteCluster remove cluster from config
func (config *HetznerConfig) DeleteCluster(name string) error {

	index, _ := config.FindClusterByName(name)

	if index == -1 {
		return errors.New("cluster not found")
	}

	config.Clusters = append(config.Clusters[:index], config.Clusters[index+1:]...)

	return nil
}

//FindClusterByName find a cluster by name in config
func (config *HetznerConfig) FindClusterByName(name string) (int, *clustermanager.Cluster) {
	for i, cluster := range config.Clusters {
		if cluster.Name == name {
			return i, &cluster
		}
	}

	return -1, nil
}

//SwitchContextByName switch to context with a specific name in app
func (app *AppConfig) SwitchContextByName(name string) error {
	ctx, err := app.FindContextByName(name)

	if err != nil {
		return err
	}

	app.CurrentContext = ctx
	app.Config.ActiveContextName = ctx.Name

	opts := []hcloud.ClientOption{
		hcloud.WithToken(ctx.Token),
	}

	app.Client = hcloud.NewClient(opts...)

	return nil
}

//FindContextByName find a context using name
func (app *AppConfig) FindContextByName(name string) (*HetznerContext, error) {

	for _, ctx := range app.Config.Contexts {
		if ctx.Name == name {

			return &ctx, nil
		}
	}

	return nil, fmt.Errorf("context '%s' not found", name)
}

// DeleteContextByName deletes a context by name from the current config
func (app *AppConfig) DeleteContextByName(name string) error {

	for idx, ctx := range app.Config.Contexts {
		if ctx.Name == name {
			app.Config.Contexts = append(app.Config.Contexts[:idx], app.Config.Contexts[idx+1:]...)
			return nil
		}
	}

	return fmt.Errorf("context '%s' not found", name)
}

func (app *AppConfig) assertActiveContext() error {
	if app.CurrentContext == nil {
		return errors.New("no context selected")
	}
	return nil
}

func makeConfigIfNotExists(appConf *AppConfig) {
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

		appConf.Config = new(HetznerConfig)
		appConf.Config.WriteCurrentConfig()

	} else {
		configFileContent, err := ioutil.ReadFile(configFileName)

		if err != nil {
			log.Fatal(err)
		}

		json.Unmarshal(configFileContent, &appConf.Config)
		if appConf.Config.ActiveContextName > "" {
			if err := appConf.SwitchContextByName(appConf.Config.ActiveContextName); err != nil {
				log.Fatal(err)
			}
		}
	}
}
