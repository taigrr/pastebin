// Package client provides a programmatic client for the pastebin service.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	formContentType = "application/x-www-form-urlencoded"
	formFieldBlob   = "blob"
)

// Client is a pastebin API client.
type Client struct {
	serviceURL string
	insecure   bool
	output     io.Writer
}

// NewClient creates a new pastebin Client.
// When insecure is true, TLS certificate verification is skipped.
// Output is written to stdout by default; use WithOutput to change.
func NewClient(serviceURL string, insecure bool) *Client {
	return &Client{serviceURL: serviceURL, insecure: insecure, output: os.Stdout}
}

// WithOutput sets the writer where paste URLs are printed.
// Returns the Client for chaining.
func (client *Client) WithOutput(writer io.Writer) *Client {
	client.output = writer
	return client
}

// Paste reads from body and submits it as a new paste.
// It prints the resulting paste URL to the configured output writer.
func (client *Client) Paste(body io.Reader) error {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: client.insecure}, //nolint:gosec // user-requested skip
	}
	httpClient := &http.Client{
		Transport: transport,
		// Don't follow redirects; capture the URL from the response.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var builder strings.Builder
	if _, err := io.Copy(&builder, body); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	formValues := url.Values{}
	formValues.Set(formFieldBlob, builder.String())

	resp, err := httpClient.PostForm(client.serviceURL, formValues)
	if err != nil {
		return fmt.Errorf("posting paste to %s: %w", client.serviceURL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Plain text response contains the paste URL in the body.
		responseBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("reading response body: %w", readErr)
		}
		fmt.Fprint(client.output, string(responseBody))
		return nil
	case http.StatusFound, http.StatusMovedPermanently:
		// HTML response redirects to the paste URL.
		location := resp.Header.Get("Location")
		if location == "" {
			return fmt.Errorf("redirect response missing Location header")
		}
		fmt.Fprint(client.output, location)
		return nil
	default:
		return fmt.Errorf("unexpected response from %s: %d", client.serviceURL, resp.StatusCode)
	}
}
