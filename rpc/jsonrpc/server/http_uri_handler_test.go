package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/line/ostracon/libs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeHTTPHandler(t *testing.T) {
	handlerFunc := makeHTTPHandler(TestRPCFunc, log.TestingLogger())
	req, _ := http.NewRequest("GET", "http://localhost/", strings.NewReader(TestGoodBody))
	rec := httptest.NewRecorder()
	handlerFunc(rec, req)
	res := rec.Result()
	require.Equal(t, http.StatusOK, res.StatusCode)
	res.Body.Close()
}

func TestMakeHTTPHandler_WS_WriteRPCResponseHTTPError_error(t *testing.T) {
	handlerFunc := makeHTTPHandler(TestWSRPCFunc, log.TestingLogger())
	req, _ := http.NewRequest("GET", "http://localhost/", nil)
	rec := NewFailedWriteResponseWriter()
	handlerFunc(rec, req)
	assert.Equal(t,
		strconv.Itoa(http.StatusNotFound),
		rec.Header().Get(http.StatusText(http.StatusNotFound)))
}

func TestMakeHTTPHandler_httpParamsToArgs_WriteRPCResponseHTTPError_error(t *testing.T) {
	handlerFunc := makeHTTPHandler(TestRPCFunc, log.TestingLogger())
	// httpParamsToArgs error
	req, _ := http.NewRequest("GET", "http://localhost/c?s=1", nil)
	// WriteRPCResponseHTTPError error
	rec := NewFailedWriteResponseWriter()
	handlerFunc(rec, req)
	assert.Equal(t,
		strconv.Itoa(http.StatusInternalServerError),
		rec.Header().Get(http.StatusText(http.StatusInternalServerError)))
}

func TestMakeHTTPHandler_unreflectResult_WriteRPCResponseHTTPError_error(t *testing.T) {
	// unreflectResult error
	handlerFunc := makeHTTPHandler(TestRPCErrorFunc, log.TestingLogger())
	req, _ := http.NewRequest("GET", "http://localhost/", nil)
	// WriteRPCResponseHTTPError error
	rec := NewFailedWriteResponseWriter()
	handlerFunc(rec, req)
	assert.Equal(t,
		strconv.Itoa(http.StatusInternalServerError),
		rec.Header().Get(http.StatusText(http.StatusInternalServerError)))
}

func TestMakeHTTPHandler_last_WriteRPCResponseHTTP_error(t *testing.T) {
	handlerFunc := makeHTTPHandler(TestRPCFunc, log.TestingLogger())
	req, _ := http.NewRequest("GET", "http://localhost/", strings.NewReader(TestGoodBody))
	// WriteRPCResponseHTTP error
	rec := NewFailedWriteResponseWriter()
	handlerFunc(rec, req)
	assert.Equal(t,
		strconv.Itoa(http.StatusOK),
		rec.Header().Get(http.StatusText(http.StatusOK)))
}
