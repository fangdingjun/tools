package main

import (
	"fmt"
	"github.com/fangdingjun/socks-go"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"strings"
)

func handleLocalForward(f string, client *ssh.Client) {
	ss := strings.FieldsFunc(f, func(c rune) bool {
		if c == ':' {
			return true
		}
		return false
	})
	if len(ss) != 4 && len(ss) != 3 {
		log.Fatal("forward addr error, format local_host:local_port:remote_host:remote_port")
	}
	var local, remote string
	if len(ss) == 4 {
		local = strings.Join(ss[:2], ":")
		remote = strings.Join(ss[2:], ":")
	} else {
		local = fmt.Sprintf(":%s", ss[0])
		remote = strings.Join(ss[1:], ":")
	}

	l, err := net.Listen("tcp", local)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan net.Conn, 5)

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				log.Println(err)
				break
			}
			ch <- c
		}
	}()

	for {
		select {
		case c := <-ch:
			go func(c net.Conn) {
				defer c.Close()
				conn, err := client.Dial("tcp", remote)
				if err != nil {
					log.Println(err)
					return
				}
				defer conn.Close()
				go io.Copy(conn, c)
				io.Copy(c, conn)
			}(c)
		case <-exitCh:
			log.Printf("exit signal received, exit...\r\n")
			l.Close()
			waitGroup.Done()
			return
		}
	}
}

func handleRemoteForward(f string, client *ssh.Client) {
	ss := strings.FieldsFunc(f, func(c rune) bool {
		if c == ':' {
			return true
		}
		return false
	})
	if len(ss) != 4 && len(ss) != 3 {
		log.Fatal("forward addr error, format remote_host:remote_port:local_host:local_port")
	}
	var local, remote string
	if len(ss) == 4 {
		remote = strings.Join(ss[:2], ":")
		local = strings.Join(ss[2:], ":")
	} else {
		remote = fmt.Sprintf(":%s", ss[0])
		local = strings.Join(ss[1:], ":")
	}
	l, err := client.Listen("tcp", remote)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan net.Conn, 5)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				log.Println(err)
				break
			}
			ch <- c
		}
	}()

	for {
		select {
		case c := <-ch:
			go func(c net.Conn) {
				defer c.Close()
				conn, err := net.Dial("tcp", local)
				if err != nil {
					log.Println(err)
					return
				}
				defer conn.Close()
				go io.Copy(conn, c)
				io.Copy(c, conn)
			}(c)
		case <-exitCh:
			log.Printf("exit signal received, exit...\r\n")
			l.Close()
			waitGroup.Done()
			return
		}
	}
}

func handleDynamicForward(f string, client *ssh.Client) {
	ss := strings.FieldsFunc(f, func(c rune) bool {
		if c == ':' {
			return true
		}
		return false
	})

	if len(ss) != 1 && len(ss) != 2 {
		log.Fatal("addr error, format [local_host:]local_port")
	}

	local := f
	if len(ss) == 1 {
		local = fmt.Sprintf(":%s", f)
	}

	l, err := net.Listen("tcp", local)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan net.Conn, 5)

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			ch <- c
		}
	}()

	for {
		select {
		case c := <-ch:
			//log.Printf("accept from %s\r\n", c.RemoteAddr())
			go func(c net.Conn) {
				s := socks.SocksConn{c, client.Dial}
				s.Serve()
			}(c)
		case <-exitCh:
			log.Printf("exit signal received, exit...\r\n")
			l.Close()
			waitGroup.Done()
			return
		}
	}
}
