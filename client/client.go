package client

import (
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
)

func InitClient(ctx context.Context) error {
	address := flag.String("l", ":1981", "Address to listen on")
	forward_to := flag.String("h", ":8080", "Remote address")
	server_name := flag.String("name", "example.com", "SNI")

	flag.Parse()

	listener, err := net.Listen("tcp", *address)

	if err != nil {
		log.Printf("Failed to listen on %s, \n%+v\n", *address, err)
		return err
	}

	return HandleTCP(listener, ctx, *forward_to, *server_name)
}

func HandleTCP(listener net.Listener, ctx context.Context, forward_to string, server_name string) error {
	go func() {
		// Clean up when context is canceled is done
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()

		if err != nil {
			return err
		}

		log.Println("connection accepted")

		go HandleTCPConn(conn, forward_to, server_name)
	}
}

func HandleTCPConn(src net.Conn, dest string, server_name string) {
	dst, err := tls.Dial("tcp", dest, &tls.Config{InsecureSkipVerify: true, ServerName: server_name})
	if err != nil {
		log.Println("Dial Error:" + err.Error())
		return
	}

	done := make(chan struct{})

	go func() {
		defer src.Close()
		defer dst.Close()
		io.Copy(dst, src)
		done <- struct{}{}
	}()

	go func() {
		defer src.Close()
		defer dst.Close()
		io.Copy(src, dst)
		done <- struct{}{}
	}()

	<-done
	<-done
}
