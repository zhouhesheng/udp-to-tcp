package client

import (
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"time"
)

func InitClient(ctx context.Context) error {
	port := flag.String("l", ":8888", "Port to listen on")
	remote := flag.String("h", ":9999", "Remote address")
	server_name := flag.String("name", "example.com", "SNI")
	is_udp := flag.Bool("u", true, "If the remote forwarded port is UDP set this")

	flag.Parse()

	if *is_udp {
		addr, err := net.ResolveUDPAddr("udp4", *port)

		if err != nil {
			log.Printf("Failed to listen on %s, \n%+v\n", *port, err)
			return err
		}

		conn, err := net.ListenUDP("udp", addr)

		if err != nil {
			log.Printf("Failed to listen on %s, \n%+v\n", *port, err)
			return err
		}

		return HandleUDPConn(conn, ctx, *remote, *server_name)
	}

	listener, err := net.Listen("tcp", *port)

	if err != nil {
		log.Printf("Failed to listen on %s, \n%+v\n", *port, err)
		return err
	}

	return HandleTCP(listener, ctx, *remote, *server_name)
}

func HandleTCP(listener net.Listener, ctx context.Context, remote string, server_name string) error {
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

		go HandleTCPConn(conn, remote, server_name)
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

func HandleUDPConn(src *net.UDPConn, ctx context.Context, remote string, server_name string) error {
redial:
	dst, err := tls.Dial("tcp", remote, &tls.Config{InsecureSkipVerify: true, ServerName: server_name})

	if err != nil {
		log.Println("Dial Error:" + err.Error())
		time.Sleep(time.Millisecond * 200)
		goto redial
	}

	done := make(chan struct{})

	go func() {
		defer src.Close()
		defer dst.Close()

		buf := make([]byte, 65507)

		for {
			n, addr, err := src.ReadFromUDP(buf)
			log.Println(addr)

			if err != nil {
				log.Println("119", err)
				break
			}

			if _, err := dst.Write(buf[:n]); err != nil {
				log.Println(err)
				break
			}
		}

		done <- struct{}{}
	}()

	go func() {
		defer src.Close()
		defer dst.Close()

		buf := make([]byte, 65507)

		for {
			n, err := dst.Read(buf)

			log.Print(n)
			if err != nil {
				log.Println("142", err)
				break
			}

			_, err2 := src.WriteTo(buf[:n], src.LocalAddr())

			if err2 != nil {
				log.Println(err)
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
	<-done
	return nil
}
