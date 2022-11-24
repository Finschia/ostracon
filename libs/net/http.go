package net

import (
	"context"
	"net/http"
	"time"
)

// HttpGet sends a GET request to the specified url with timeout and return the response.
func HttpGet(url string, timeout time.Duration) (*http.Response, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return HttpRequest(request, timeout)
}

// HttpRequest sends the specified HTTP requests with timeout and return the response.
// For stability and security reason, we need to be available request timeouts, so http.Client{} and http.Get() are
// overridden with functions that rely on this function.
func HttpRequest(request *http.Request, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{
		Timeout: timeout,
	}
	return client.Do(request)
}
