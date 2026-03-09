// Package client provides a programmatic client for the pastebin service.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	formContentType = "application/x-www-form-urlencoded"
	formFieldBlob   = "blob"
)

// Client is a pastebin API client.
type Client struct {
	url      string
	insecure bool
}

// NewClient creates a new pastebin Client.
// When insecure is true, TLS certificate verification is skipped.
func NewClient(serviceURL string, insecure bool) *Client {
	return &Client{url: serviceURL, insecure: insecure}
}

// Paste reads from body and submits it as a new paste.
// It prints the resulting paste URL to stdout.
func (c *Client) Paste(body io.Reader) error {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecure}, //nolint:gosec // user-requested skip
	}
	httpClient := &http.Client{Transport: transport}

	var builder strings.Builder
	if _, err := io.Copy(&builder, body); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	formValues := url.Values{}
	formValues.Set(formFieldBlob, builder.String())

	resp, err := httpClient.PostForm(c.url, formValues)
	if err != nil {
		return fmt.Errorf("posting paste to %s: %w", c.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMovedPermanently {
		return fmt.Errorf("unexpected response from %s: %d", c.url, resp.StatusCode)
	}

	fmt.Print(resp.Request.URL.String())
	return nil
}
