package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"reflect"

	ocabci "github.com/line/ostracon/abci/types"
	tmnet "github.com/line/ostracon/libs/net"
)

func main() {

	conn, err := tmnet.Connect("unix://test.sock")
	if err != nil {
		log.Fatal(err.Error())
	}

	// Make a bunch of requests
	counter := 0
	for i := 0; ; i++ {
		req := ocabci.ToRequestEcho("foobar")
		_, err := makeRequest(conn, req)
		if err != nil {
			log.Fatal(err.Error())
		}
		counter++
		if counter%1000 == 0 {
			fmt.Println(counter)
		}
	}
}

func makeRequest(conn io.ReadWriter, req *ocabci.Request) (*ocabci.Response, error) {
	var bufWriter = bufio.NewWriter(conn)

	// Write desired request
	err := ocabci.WriteMessage(req, bufWriter)
	if err != nil {
		return nil, err
	}
	err = ocabci.WriteMessage(ocabci.ToRequestFlush(), bufWriter)
	if err != nil {
		return nil, err
	}
	err = bufWriter.Flush()
	if err != nil {
		return nil, err
	}

	// Read desired response
	var res = &ocabci.Response{}
	err = ocabci.ReadMessage(conn, res)
	if err != nil {
		return nil, err
	}
	var resFlush = &ocabci.Response{}
	err = ocabci.ReadMessage(conn, resFlush)
	if err != nil {
		return nil, err
	}
	if _, ok := resFlush.Value.(*ocabci.Response_Flush); !ok {
		return nil, fmt.Errorf("expected flush response but got something else: %v", reflect.TypeOf(resFlush))
	}

	return res, nil
}
