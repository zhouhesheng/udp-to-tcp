package main

import (
	"context"
	"log"
	"os"

	"github.com/stargazer39/simple-proxy/client"
	"github.com/stargazer39/simple-proxy/server"
)

func main() {
	log.Println("Starting SIMPROX")
	main_ctx := context.TODO()
	ctx, cancel := context.WithCancel(main_ctx)

	defer cancel()

	if len(os.Args) < 2 {
		log.Println("Not enough args.")
		os.Exit(-1)
		return
	}

	mode := os.Args[1]
	os.Args = os.Args[1:]

	if mode == "client" {
		log.Println("Starting in client mode")
		client.InitClient(ctx)
	} else {
		log.Println("Starting in server mode")
		server.InitServer(ctx)
	}
}
