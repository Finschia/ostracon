package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Finschia/ostracon/libs/log"
	"github.com/Finschia/ostracon/rpc/jsonrpc/types"
)

var (
	TestJSONIntID = types.JSONRPCIntID(-1)
	TestRPCError  = &types.RPCError{}
	TestRawMSG    = json.RawMessage(`{"p1":"v1"}`)
	TestText      = "foo"
	ErrFoo        = errors.New(TestText)

	TestRPCFunc = NewRPCFunc(
		func(ctx *types.Context, s string, i int) (string, error) { return TestText, nil }, "s,i")
	TestRPCErrorFunc = NewRPCFunc(
		func(ctx *types.Context, s string, i int) (string, error) { return "", ErrFoo }, "s,i")
	TestWSRPCFunc = NewWSRPCFunc(
		func(ctx *types.Context, s string, i int) (string, error) { return TestText, nil }, "s,i")

	TestFuncMap            = map[string]*RPCFunc{"c": TestRPCFunc}
	TestGoodBody           = `{"jsonrpc": "2.0", "method": "c", "id": "0", "params": null}`
	TestBadParams          = `{"jsonrpc": "2.0", "method": "c", "id": "0", "params": "s=a,i=b"}`
	TestMaxBatchRequestNum = "10"
)

type FailManager struct {
	counter       int
	failedCounter int
	throwPanic    bool
}

func (fm *FailManager) checkAndDo(
	encounter func() (int, error),
	throwing func(),
) (int, error) {
	if fm.counter == fm.failedCounter {
		fmt.Println("FailManager:do encounter")
		return encounter()
	}
	fm.counter++
	if fm.throwPanic {
		fmt.Println("FailManager:do throwing")
		throwing()
	}
	return 0, nil
}

type FailedWriteResponseWriter struct {
	header http.Header
	fm     *FailManager
	code   int
	error  error
}

func NewFailedWriteResponseWriter() FailedWriteResponseWriter {
	return FailedWriteResponseWriter{
		header: make(http.Header),
		fm:     &FailManager{},
		code:   -1,
		error:  fmt.Errorf("error"),
	}
}
func (frw FailedWriteResponseWriter) Header() http.Header {
	return frw.header
}
func (frw FailedWriteResponseWriter) Write(buf []byte) (int, error) {
	fmt.Println("FailedWriteResponseWriter:" + strconv.Itoa(frw.fm.counter) + ":" + string(buf))
	return frw.fm.checkAndDo(
		func() (int, error) {
			return frw.code, frw.error
		},
		func() {
			res := types.RPCResponse{}
			res.UnmarshalJSON(buf) // nolint: errcheck
			panic(res)
		},
	)
}
func (frw FailedWriteResponseWriter) WriteHeader(code int) {
	frw.header.Set(http.StatusText(code), strconv.Itoa(code))
}

type FailedLogger struct {
	fm *FailManager
}

func NewFailedLogger() FailedLogger {
	return FailedLogger{
		fm: &FailManager{},
	}
}
func (l *FailedLogger) Info(msg string, keyvals ...interface{}) {
	fmt.Println("FailedLogger.Info:" + msg)
}
func (l *FailedLogger) Debug(msg string, keyvals ...interface{}) {
	fmt.Println("FailedLogger.Debug:" + msg)
}
func (l *FailedLogger) Error(msg string, keyvals ...interface{}) {
	fmt.Println("FailedLogger.Error:" + strconv.Itoa(l.fm.counter) + ":" + msg)
	l.fm.checkAndDo( // nolint: errcheck
		func() (int, error) { panic(l.fm.counter) },
		func() {},
	)
}
func (l *FailedLogger) With(keyvals ...interface{}) log.Logger {
	return l
}
