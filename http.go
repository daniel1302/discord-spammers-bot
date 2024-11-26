package main

import (
	"net/http"
	"time"
)

func DefaultHttpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   timeout / 3,
			ResponseHeaderTimeout: timeout - (timeout / 3),
		},
	}
}
