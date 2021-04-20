package main

import (
	"log"
	"os"

	"github.com/prologic/pastebin/client"

	"github.com/namsral/flag"
)

const (
	defaultConfig     = "pastebin.conf"
	defaultUserConfig = "~/.pastebin.conf"
	defaultURL        = "http://localhost:8000"
)

func main() {
	var (
		url      string
		insecure bool
	)

	flag.StringVar(&url, "url", defaultURL, "pastebin service url")
	flag.BoolVar(&insecure, "insecure", false, "insecure (skip ssl verify)")

	flag.Parse()

	cli := client.NewClient(url, insecure)

	err := cli.Paste(os.Stdin)
	if err != nil {
		log.Printf("error posting paste: %s", err)
		os.Exit(1)
	}
}
