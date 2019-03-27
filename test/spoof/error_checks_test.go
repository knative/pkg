package spoof

import (
	"net/http"
	"testing"
)

func TestDNSError(t *testing.T) {
	client := &http.Client{}

	for _, tt := range []struct {
		name     string
		url      string
		dnsError bool
	}{{
		name:     "url does not exist",
		url:      "http://this.url.does.not.exist",
		dnsError: true,
	}, {
		name:     "ip address",
		url:      "http://127.0.0.1",
		dnsError: false,
	}, {
		name:     "localhost",
		url:      "http://localhost:8080",
		dnsError: false,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			_, err := client.Do(req)
			if dnsError := isDNSError(err); tt.dnsError != dnsError {
				t.Errorf("Expected dnsError=%v, got %v", tt.dnsError, dnsError)
			}
		})
	}
}
