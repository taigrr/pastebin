package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/taigrr/jety"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	var configFile string

	InitConfig()

	rootCmd := &cobra.Command{
		Use:   "pastebin",
		Short: "A self-hosted ephemeral pastebin service",
		Long:  "pastebin is a self-hosted pastebin web app that lets you create and share ephemeral data between devices and users.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := ReadConfigFile(configFile); err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			// Override jety values with any flags explicitly set on the command line.
			cmd.Flags().Visit(func(flag *pflag.Flag) {
				jety.SetString(flag.Name, flag.Value.String())
			})
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig()
			if cfg.Expiry.Seconds() < 60 {
				return fmt.Errorf("expiry of %s is too small (minimum 1m)", cfg.Expiry)
			}
			server := NewServer(cfg)
			return server.ListenAndServe()
		},
	}

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "path to config file (JSON, TOML, or YAML)")
	rootCmd.Flags().String(configKeyBind, defaultBind, "address and port to bind to")
	rootCmd.Flags().String(configKeyFQDN, defaultFQDN, "FQDN for public access")
	rootCmd.Flags().Duration(configKeyExpiry, defaultExpiry, "expiry time for pastes")

	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}
