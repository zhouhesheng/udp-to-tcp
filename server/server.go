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
	is_udp := flag.Bool("u", false, "If the remote forwarded port is UDP set this")

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

	if *is_udp {
		addr, err := net.ResolveUDPAddr("udp", *forward_to)

		if err != nil {
			log.Fatal("Wrong UDP address", err.Error())
		}

		*forward_to = addr.String()
	}

	return HandleTCP(listener, ctx, *forward_to, *is_udp)
}

func HandleTCP(listener net.Listener, ctx context.Context, forward_to string, is_udp bool) error {
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

			if is_udp {
				go HandleUDPConnection(conn, forward_to)
				continue
			}

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

func HandleUDPConnection(src net.Conn, dest string) {
	addr, err := net.ResolveUDPAddr("udp", dest)

	if err != nil {
		log.Println("Resolve Error:" + err.Error())
		return
	}

	dst, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		log.Println("Dial Error:" + err.Error())
		return
	}

	done := make(chan struct{})

	go func() {
		defer src.Close()
		defer dst.Close()

		buf := make([]byte, 65507)

		for {
			n, addr, err := dst.ReadFromUDP(buf)
			log.Println(addr)

			if err != nil {
				log.Println(err)
				break
			}

			if _, err := src.Write(buf[:n]); err != nil {
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
			n, err := src.Read(buf)

			if err != nil {
				log.Println(err)
				break
			}

			_, err2 := dst.WriteTo(buf[:n], addr)

			if err2 != nil {
				log.Println(err)
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
	<-done

}
