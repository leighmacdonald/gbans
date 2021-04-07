package util

import (
	"net/http"
	"time"
)

func NewHTTPClient(timeout time.Duration) *http.Client {
	c := &http.Client{
		Timeout: timeout,
	}
	return c
}
