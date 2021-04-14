package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type errorResponse struct {
	Errors []struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
		Source struct {
			Pointer string `json:"pointer"`
		} `json:"source"`
		Status string `json:"status"`
		Title  string `json:"title"`
	} `json:"errors"`
}

type successResponse struct {
	Data  interface{}       `json:"data"`
	Links map[string]string `json:"links"`
}

// Client used to communicate with the API
type Client struct {
	Host       string
	Token      string
	HTTPClient *http.Client
}

type withHeaderTransport struct {
	Hostname string
	http.Header
	rt http.RoundTripper
}

func createWithHeaderTransport(rt http.RoundTripper, hostname string) withHeaderTransport {
	return withHeaderTransport{Hostname: hostname, Header: make(http.Header), rt: rt}
}

func (h withHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if h.Hostname == req.URL.Hostname() {
		for header, value := range h.Header {
			if _, hasHeader := req.Header[header]; !hasHeader {
				req.Header[header] = value
			}
		}
	}
	return h.rt.RoundTrip(req)
}

type ExternalRedirectError struct {
	URL url.URL
}

func (m *ExternalRedirectError) Error() string {
	return "ExternalRedirect"
}

// NewClient creates a new Client
func NewClient(token string, host string) *Client {
	rt := http.DefaultTransport.(*http.Transport).Clone()

	// certPEM := []byte("xxxx") // load from file https://raw.githubusercontent.com/transifex/transifex-client/master/txclib/cacert.pem
	// certPool := x509.NewCertPool()
	// if !certPool.AppendCertsFromPEM(certPEM) {
	// 	log.Fatal("AppendCertsFromPEM failed")
	// }
	// https://golang.org/src/crypto/x509/ check roots
	rt.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: false,
		// RootCAs:            certPool,
	}

	u, _ := url.Parse(host)
	hostname := u.Hostname()
	rt.Proxy = http.ProxyFromEnvironment
	rtWithHeaders := createWithHeaderTransport(rt, hostname)
	rtWithHeaders.Set("Authorization", "Bearer "+token)
	rtWithHeaders.Set("Content-Type", "application/vnd.api+json")
	httpClient := http.Client{
		Transport: rtWithHeaders,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if hostname == req.URL.Hostname() {
				return nil
			}
			return &ExternalRedirectError{URL: *req.URL}
		},
	}
	return &Client{
		Host:       host,
		Token:      token,
		HTTPClient: &httpClient,
	}
}

func (c *Client) DoRequest(req *http.Request, apiResource interface{}) error {
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			output, err := JSONMarshal(errRes)
			if err == nil {
				return fmt.Errorf(
					"API responded with error code \"%d\" \n%s",
					res.StatusCode,
					string(output),
				)
			}
		}
		return fmt.Errorf("unknown error, status code: %d", res.StatusCode)
	}
	fullResponse := successResponse{
		Data: apiResource,
	}
	if err = json.NewDecoder(res.Body).Decode(&fullResponse); err != nil {
		return err
	}
	return nil
}
