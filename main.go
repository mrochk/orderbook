package main

import (
	"fmt"
	"os"

	"github.com/mrochk/exchange/orderbook"
	"github.com/mrochk/exchange/server"
)

const defaultPort = "8080"

func main() {
	var (
		port      string
		orderbook = orderbook.New()
		server    = server.New(orderbook)
	)
	if len(os.Args) > 1 {
		port = os.Args[1]
	} else {
		port = defaultPort
	}
	fmt.Printf("Ready to receive requests on port %s...\n", port)
	server.Run(":" + port)
}
