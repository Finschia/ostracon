package net

import (
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHttpGet(t *testing.T) {
	expected := "hello, world"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		_, err := w.Write([]byte(expected))
		require.NoError(t, err)
	}))
	defer server.Close()

	response, err := HttpGet(server.URL, 60*time.Second)
	require.NoError(t, err)
	bytes, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, expected, string(bytes))
}

func TestHttpGetWithTimeout(t *testing.T) {
	shutdown := make(chan string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var _ = <-shutdown
	}))
	defer server.Close()

	accuracy := 0.05
	timeout := 10 * time.Second
	defer func() { shutdown <- "shutdown" }()
	t0 := time.Now()
	_, err := HttpGet(server.URL, timeout)
	t1 := time.Now()
	require.Error(t, err)
	delta := t1.Sub(t0).Seconds()
	require.Greater(t, delta, timeout.Seconds())
	require.InDeltaf(t, timeout.Seconds(), delta, accuracy,
		"response time of %.3f sec exceeded +%d%% of the expected timeout of %.3f sec", delta, uint(accuracy*100), timeout.Seconds())
}

func TestHttpGetWithInvalidURL(t *testing.T) {
	_, err := HttpGet("\n", 0*time.Second)
	require.Error(t, err)
}
