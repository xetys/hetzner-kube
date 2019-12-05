package cmd

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xetys/hetzner-kube/pkg"
)

var cfgFile string
var DebugMode bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hetzner-kube",
	Short: "A CLI tool to provision kubernetes clusters on Hetzner Cloud",
	Long: `A tool for creating and managing kubernetes clusters on Hetzner Cloud.

	`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		pkg.RenderProgressBars = false
		if DebugMode {
			fmt.Println("Running in Debug Mode!")
			pkg.RenderProgressBars = true
		}
		AppConf = NewAppConfig(DebugMode)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file to use")
	rootCmd.PersistentFlags().BoolVarP(&DebugMode, "debug", "d", false, "debug mode")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		setConfigDirectory()
	}

	// read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func setConfigDirectory() {
	// Find config dir based on XDG Base Directory Specification
	// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		viper.AddConfigPath(xdgConfig)
	}

	// Failback to home directory
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
	}

	if err == nil {
		viper.AddConfigPath(home)
	}

	if xdgConfig == "" && err != nil {
		fmt.Println("Unable to detect any config location, please specify it with --config flag")
		os.Exit(1)
	}

	// Search config directory with name ".hetzner-kube" (without extension).
	viper.SetConfigName(".hetzner-kube")
}
