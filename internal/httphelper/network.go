package httphelper

import (
	"net/http"
	"time"
)

// NewHTTPClient allocates a preconfigured *http.Client.
func NewHTTPClient() *http.Client {
	c := &http.Client{
		Timeout: time.Second * 10,
	}

	return c
}
