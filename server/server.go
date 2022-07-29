package server

import (
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
)

func InitServer(ctx context.Context) error {
	address := flag.String("l", ":8080", "Address to listen on")
	forward_to := flag.String("f", ":1985", "Forward address")

	flag.Parse()

	cert, key := GenRandomCert()

	tls_cert, err := tls.X509KeyPair(cert, key)

	if err != nil {
		log.Fatal("Cannot be loaded the certificate.", err.Error())
	}

	listener, err := tls.Listen("tcp", *address, &tls.Config{Certificates: []tls.Certificate{tls_cert}})

	if err != nil {
		log.Fatal("Can't listen on port specified.", err.Error())
	}

	return HandleTCP(listener, ctx, *forward_to)
}

func HandleTCP(listener net.Listener, ctx context.Context, forward_to string) error {
	for {
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

			go HandleTCPConn(conn, forward_to)
		}
	}
}

func HandleTCPConn(src net.Conn, dest string) {
	dst, err := net.Dial("tcp", dest)
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
