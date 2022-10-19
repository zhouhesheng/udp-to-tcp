package server

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
)

func InitServer(ctx context.Context) error {
	address := flag.String("l", ":9999", "Address to listen on")
	forward_to := flag.String("f", "1.1.1.1:53", "Forward address")
	is_udp := flag.Bool("u", true, "If the remote forwarded port is UDP set this")

	flag.Parse()

	addr, err := net.ResolveTCPAddr("tcp", *address)
	if err != nil {
		log.Printf("Unable to resolve IP")
	}

	listener, err := net.ListenTCP("tcp", addr)

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

func HandleTCP(listener *net.TCPListener, ctx context.Context, forward_to string, is_udp bool) error {
	for {
		go func() {
			// Clean up when context is canceled is done
			<-ctx.Done()
			listener.Close()
		}()

		for {
			conn, err := listener.AcceptTCP()

			if err != nil {
				log.Println("HandleTCP err", err)
				return err
			}

			log.Println("connection accepted")

			if is_udp {
				err := conn.SetKeepAlive(true)
				if err != nil {
					log.Printf("Unable to set keepalive - %s", err)
				}
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

		buf := make([]byte, 65535)

		for {
			n, err := dst.Read(buf)
			if err != nil {
				log.Println("read udp error", err)
				break
			}

			if _, err := src.Write(buf[:n]); err != nil {
				log.Println("write tcp error", err)
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		defer src.Close()
		defer dst.Close()

		buf := make([]byte, 65535)

		for {
			n, err := src.Read(buf)
			log.Println("read tcp n=", n)

			if err != nil {
				log.Println("read tcp error", err)
				break
			}

			m, err2 := dst.Write(buf[0:n])
			log.Println("write dst ", addr, m)
			if err2 != nil {
				log.Println("write dst error=", err)
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
	<-done

}
