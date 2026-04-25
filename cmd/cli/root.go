package main

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "casper",
	Short: "CasperCloud API CLI",
	Long:  "Command-line client for the CasperCloud REST API.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		printErr(err.Error())
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("api-url", "", "API base URL (overrides config; default from ~/.casper/config.json or http://127.0.0.1:8080)")
	_ = viper.BindPFlag("api_base_url", rootCmd.PersistentFlags().Lookup("api-url"))

	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(instancesCmd())
	rootCmd.AddCommand(volumesCmd())
	rootCmd.AddCommand(networksCmd())
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	viper.AddConfigPath(filepath.Join(home, ".casper"))
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.SetEnvPrefix("CASPER")
	viper.AutomaticEnv()
	_ = viper.BindEnv("api_base_url", "CASPER_API_URL")
	_ = viper.ReadInConfig()
}

func apiBaseURL() (string, error) {
	u := viper.GetString("api_base_url")
	if u == "" {
		u = "http://127.0.0.1:8080"
	}
	return u, nil
}

func mustClient() *Client {
	base, err := apiBaseURL()
	if err != nil {
		printErr(err.Error())
		os.Exit(1)
	}
	c, err := NewClient(ClientOptions{
		BaseURL:   base,
		Token:     viper.GetString("token"),
		ProjectID: viper.GetString("project_id"),
	})
	if err != nil {
		printErr(err.Error())
		os.Exit(1)
	}
	return c
}

func requireProjectAuth() *Client {
	c := mustClient()
	if c.Token == "" || c.ProjectID == uuid.Nil {
		printErr("not logged in or no active project: run `casper auth login`")
		os.Exit(1)
	}
	return c
}
