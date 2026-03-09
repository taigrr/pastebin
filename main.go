package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	var cfg Config

	rootCmd := &cobra.Command{
		Use:   "pastebin",
		Short: "A self-hosted ephemeral pastebin service",
		Long:  "pastebin is a self-hosted pastebin web app that lets you create and share ephemeral data between devices and users.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Expiry.Seconds() < 60 {
				return fmt.Errorf("expiry of %s is too small (minimum 1m)", cfg.Expiry)
			}
			server := NewServer(cfg)
			return server.ListenAndServe()
		},
	}

	rootCmd.Flags().StringVar(&cfg.Bind, "bind", defaultBind, "address and port to bind to")
	rootCmd.Flags().StringVar(&cfg.FQDN, "fqdn", defaultFQDN, "FQDN for public access")
	rootCmd.Flags().DurationVar(&cfg.Expiry, "expiry", defaultExpiry, "expiry time for pastes")

	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}

const (
	defaultBind   = "0.0.0.0:8000"
	defaultFQDN   = "localhost"
	defaultExpiry = 5 * time.Minute
)
