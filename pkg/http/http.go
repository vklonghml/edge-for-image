package http

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/golang/glog"
)

const (
	defaultConnectTimeout            = 30 * time.Second
	defaultKeepAliveTimeout          = 30 * time.Second
	defaultResponseReadTimeout       = 300 * time.Second
	defaultMaxIdleConnectionsPerHost = 3
)

var (
	connectionTimeout         = defaultConnectTimeout
	keepaliveTimeout          = defaultKeepAliveTimeout
	responseReadTimeout       = defaultResponseReadTimeout
	maxIdleConnectionsPerHost = defaultMaxIdleConnectionsPerHost
)

// NewHTTPClient create new http client
func NewHTTPClient() *http.Client {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   connectionTimeout,
			KeepAlive: keepaliveTimeout,
		}).Dial,
		MaxIdleConnsPerHost:   maxIdleConnectionsPerHost,
		ResponseHeaderTimeout: responseReadTimeout,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	glog.Infof("tlsconfig insecureskipverify true")
	return &http.Client{Transport: transport}
}

// SendRequest send http request
func SendRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	_, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, err
	}
	// glog.V(2).Infof("POST request: %s", string(body))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// BuildRequest create a http request
func BuildRequest(method, urlStr string, body io.Reader, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Add("Content-Type", "application/json")
	}
	return req, nil
}
