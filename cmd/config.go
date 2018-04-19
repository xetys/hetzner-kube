package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log/term"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/thcyron/uiprogress"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

var DefaultConfigPath string
var AppConf AppConfig = AppConfig{}

type AppSSHClient struct {
}

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
}

func (config *HetznerConfig) AddContext(context HetznerContext) {
	config.Contexts = append(config.Contexts, context)
}

func (config *HetznerConfig) AddSSHKey(key clustermanager.SSHKey) {
	config.SSHKeys = append(config.SSHKeys, key)
}

func (config *HetznerConfig) DeleteSSHKey(name string) error {

	index, _ := config.FindSSHKeyByName(name)

	if index == -1 {
		return errors.New("ssh key not found")
	}

	config.SSHKeys = append(config.SSHKeys[:index], config.SSHKeys[index+1:]...)

	return nil
}

func (config *HetznerConfig) FindSSHKeyByName(name string) (int, *clustermanager.SSHKey) {
	index := -1
	for i, v := range config.SSHKeys {
		if v.Name == name {
			index = i
			return index, &v
		}
	}
	return index, nil
}

func (config *HetznerConfig) AddCluster(cluster clustermanager.Cluster) {
	for i, v := range config.Clusters {
		if v.Name == cluster.Name {
			config.Clusters[i] = cluster
			return
		}
	}

	config.Clusters = append(config.Clusters, cluster)
}

func (config *HetznerConfig) DeleteCluster(name string) error {

	index, _ := config.FindClusterByName(name)

	if index == -1 {
		return errors.New("cluster not found")
	}

	config.Clusters = append(config.Clusters[:index], config.Clusters[index+1:]...)

	return nil
}

func (config *HetznerConfig) FindClusterByName(name string) (int, *clustermanager.Cluster) {
	for i, cluster := range config.Clusters {
		if cluster.Name == name {
			return i, &cluster
		}
	}

	return -1, nil
}

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

	AppConf.Client = hcloud.NewClient(opts...)

	return nil
}

func (app *AppConfig) FindContextByName(name string) (*HetznerContext, error) {

	for _, ctx := range app.Config.Contexts {
		if ctx.Name == name {

			return &ctx, nil
		}
	}

	return nil, fmt.Errorf("context '%s' not found", name)
}

func (app *AppConfig) DeleteContextByName(name string) (error) {

	for idx, ctx := range app.Config.Contexts {
		if ctx.Name == name {
			app.Config.Contexts = append(app.Config.Contexts[:idx], app.Config.Contexts[idx+1:]...)
			return nil
		}
	}

	return fmt.Errorf("context '%s' not found", name)
}

// deprecated
func (app *AppConfig) ActionProgress(ctx context.Context, action *hcloud.Action) error {
	errCh, progressCh := waitAction(ctx, app.Client, action)

	if term.IsTerminal(os.Stdout) {
		progress := uiprogress.New()

		progress.Start()
		bar := progress.AddBar(100).AppendCompleted().PrependElapsed()
		bar.Empty = ' '

		for {
			select {
			case err := <-errCh:
				if err == nil {
					bar.Set(100)
				}
				progress.Stop()
				return err
			case p := <-progressCh:
				bar.Set(p)
			}
		}
	} else {
		return <-errCh
	}
}

func (app *AppConfig) assertActiveContext() error {
	if app.CurrentContext == nil {
		return errors.New("no context selected")
	}
	return nil
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
	AppConf.SSHClient = clustermanager.NewSSHCommunicator(AppConf.Config.SSHKeys)
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
		if AppConf.Config.ActiveContextName > "" {
			if err := AppConf.SwitchContextByName(AppConf.Config.ActiveContextName); err != nil {
				log.Fatal(err)
			}
		}
	}
}
