package net

import (
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
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
	var mtx sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mtx.Lock()
		mtx.Unlock()
	}))
	defer server.Close()

	accuracy := 0.05
	timeout := 10 * time.Second
	mtx.Lock()
	defer mtx.Unlock()
	t0 := time.Now()
	_, err := HttpGet(server.URL, timeout)
	t1 := time.Now()
	require.Error(t, err)
	delta := t1.Sub(t0).Seconds()
	require.Greater(t, delta, timeout.Seconds())
	require.InDeltaf(t, timeout.Seconds(), delta, accuracy,
		"response time of %.3f sec exceeded +%d%% of the expected timeout of %.3f sec", delta, uint(accuracy*100), timeout.Seconds())
}
