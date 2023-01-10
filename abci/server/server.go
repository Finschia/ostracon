/*
Package server is used to start a new ABCI server.

It contains two server implementation:
 * gRPC server
 * socket server

*/
package server

import (
	"fmt"

	ocabci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/service"
)

func NewServer(protoAddr, transport string, app ocabci.Application) (service.Service, error) {
	var s service.Service
	var err error
	switch transport {
	case "socket":
		s = NewSocketServer(protoAddr, app)
	case "grpc":
		s = NewGRPCServer(protoAddr, ocabci.NewGRPCApplication(app))
	default:
		err = fmt.Errorf("unknown server type %s", transport)
	}
	return s, err
}
