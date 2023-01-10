package main

import (
	"bufio"
	"fmt"
	"log"

	ocabci "github.com/line/ostracon/abci/types"
	tmnet "github.com/line/ostracon/libs/net"
)

func main() {

	conn, err := tmnet.Connect("unix://test.sock")
	if err != nil {
		log.Fatal(err.Error())
	}

	// Read a bunch of responses
	go func() {
		counter := 0
		for {
			var res = &ocabci.Response{}
			err := ocabci.ReadMessage(conn, res)
			if err != nil {
				log.Fatal(err.Error())
			}
			counter++
			if counter%1000 == 0 {
				fmt.Println("Read", counter)
			}
		}
	}()

	// Write a bunch of requests
	counter := 0
	for i := 0; ; i++ {
		var bufWriter = bufio.NewWriter(conn)
		var req = ocabci.ToRequestEcho("foobar")

		err := ocabci.WriteMessage(req, bufWriter)
		if err != nil {
			log.Fatal(err.Error())
		}
		err = bufWriter.Flush()
		if err != nil {
			log.Fatal(err.Error())
		}

		counter++
		if counter%1000 == 0 {
			fmt.Println("Write", counter)
		}
	}
}
