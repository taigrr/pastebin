package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	contentType = "application/x-www-form-urlencoded"
)

// Client ...
type Client struct {
	url      string
	insecure bool
}

// NewClient ...
func NewClient(url string, insecure bool) *Client {
	return &Client{url: url, insecure: insecure}
}

// Paste ...
func (c *Client) Paste(body io.Reader) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !c.insecure},
	}
	client := &http.Client{Transport: tr}

	buf := new(strings.Builder)
	_, err := io.Copy(buf, body)
	// check errors
	if err != nil {
		log.Printf("error reading in file: %s", err)
		return err
	}
	v := url.Values{}
	v.Set("blob", buf.String())
	res, err := client.PostForm(c.url, v)
	if err != nil {
		log.Printf("error pasting to %s: %s", c.url, err)
		return err
	}

	if res.StatusCode != 200 && res.StatusCode != 301 {
		log.Printf("unexpected response from %s: %d", c.url, res.StatusCode)
		return errors.New("unexpected response")
	}

	fmt.Printf("%s", res.Request.URL.String())

	return nil
}
