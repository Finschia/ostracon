package client

import (
	"net/http"
	"time"
)

// HTTPClientForTest set short second on Transport.IdleConnTimeout
// See: DefaultHTTPClient
//    : Transport.IdleConnTimeout:       defaultIdleConnTimeout * time.Second
func HTTPClientForTest(remoteAddr string) (*http.Client, error) {
	dialFn, _ := makeHTTPDialer(remoteAddr)

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			// Set to true to prevent GZIP-bomb DoS attacks
			DisableCompression:    true,
			Dial:                  dialFn,
			MaxIdleConns:          defaultMaxIdleConns,
			MaxIdleConnsPerHost:   defaultMaxIdleConns,
			IdleConnTimeout:       1 * time.Second, // set short second for test
			ExpectContinueTimeout: defaultExpectContinueTimeout * time.Second,
		},
	}

	return client, nil
}
