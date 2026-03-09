package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/taigrr/pastebin/client"
)

const (
	defaultURL = "http://localhost:8000"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	var (
		serviceURL string
		insecure   bool
	)

	rootCmd := &cobra.Command{
		Use:   "pb",
		Short: "CLI client for the pastebin service",
		Long:  "pb reads from stdin and submits the content as a paste to the pastebin service.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := client.NewClient(serviceURL, insecure)
			if err := cli.Paste(os.Stdin); err != nil {
				return fmt.Errorf("posting paste: %w", err)
			}
			return nil
		},
	}

	rootCmd.Flags().StringVar(&serviceURL, "url", defaultURL, "pastebin service URL")
	rootCmd.Flags().BoolVar(&insecure, "insecure", false, "skip TLS certificate verification")

	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}
