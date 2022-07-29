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

	if os.Args[1] == "client" {
		client.InitClient(ctx)
	} else {
		server.InitServer(ctx)
	}
}
