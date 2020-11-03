package main

import (
	"fmt"
	"github.com/jaxxstorm/ploy/cmd/ploy/destroy"
	"github.com/jaxxstorm/ploy/cmd/ploy/get"
	"github.com/jaxxstorm/ploy/cmd/ploy/up"
	"github.com/jaxxstorm/ploy/pkg/contract"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var (
	org    string
	debug  bool
	region string
)

func configureCLI() *cobra.Command {
	rootCommand := &cobra.Command{
		Use:  "ploy",
		Long: "Deploy your applications",
	}

	rootCommand.AddCommand(up.Command())
	rootCommand.AddCommand(destroy.Command())
	rootCommand.AddCommand(get.Command())

	rootCommand.PersistentFlags().StringVarP(&org, "org", "o", "", "Pulumi org to use for your stack")
	rootCommand.PersistentFlags().StringVarP(&region, "region", "r", "us-west-2", "AWS Region to use")
	rootCommand.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	viper.BindEnv("region", "AWS_REGION") // if the user has set the AWS_REGION env var, use it

	viper.BindPFlag("org", rootCommand.PersistentFlags().Lookup("org"))
	viper.BindPFlag("region", rootCommand.PersistentFlags().Lookup("region"))

	return rootCommand
}

func init() {
	log.SetLevel(log.InfoLevel)
	cobra.OnInitialize(initConfig)
}

func initConfig() {

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.ploy") // adding home directory as first search path
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debug("Using config file: ", viper.ConfigFileUsed())
	}
}

func main() {
	rootCommand := configureCLI()

	if err := rootCommand.Execute(); err != nil {
		contract.IgnoreIoError(fmt.Fprintf(os.Stderr, "%s", err))
		os.Exit(1)
	}
}
