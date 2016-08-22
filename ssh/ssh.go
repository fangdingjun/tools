package main

import (
	//"bytes"
	"flag"
	//"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
	"time"
	//"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

var exitCh chan int

//var doneCh chan int
var waitGroup = &sync.WaitGroup{}

type conn struct {
	net.Conn
	timeout time.Duration
}

func (c *conn) Read(buf []byte) (n int, err error) {
	err = c.Conn.SetReadDeadline(time.Now().Add(c.timeout))
	if err != nil {
		return
	}
	return c.Conn.Read(buf)
}

func (c *conn) Write(buf []byte) (n int, err error) {
	err = c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
	if err != nil {
		return
	}
	return c.Conn.Write(buf)
}

// timeout dail method
// refer to http://stackoverflow.com/a/31566330/6734216
func sshDialTimeout(network, addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	c, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}
	timeoutConn := &conn{c, timeout}
	c1, chans, reqs, err := ssh.NewClientConn(timeoutConn, addr, config)
	if err != nil {
		return nil, err
	}

	client := ssh.NewClient(c1, chans, reqs)
	// keep alive
	go func() {
		t := time.NewTicker(timeout - time.Second*5)
		defer t.Stop()
		errCount := 0
		for {
			<-t.C
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				//log.Println(err)
				errCount++
			} else {
				errCount = 0
			}

			if errCount >= 5 {
				log.Printf("keepalive hit max count, signal exit...\r\n")
				for i := 0; i < 5; i++ {
					select {
					case exitCh <- 1:
					default:
					}
				}
				return

			}
		}
	}()
	return client, nil
}

func main() {
	var host, port, user, pass, key string
	var localForward, remoteForward, dynamicForward string
	var notRunCmd bool

	flag.StringVar(&user, "l", "", "ssh username")
	flag.StringVar(&pass, "pw", "", "ssh password")
	flag.StringVar(&port, "p", "0", "remote port")
	flag.StringVar(&key, "i", "", "private key file")
	flag.StringVar(&localForward, "L", "", "forward local port to remote, format [local_host:]local_port:remote_host:remote_port")
	flag.StringVar(&remoteForward, "R", "", "forward remote port to local, format [remote_host:]remote_port:local_host:local_port")
	flag.BoolVar(&notRunCmd, "N", false, "not run remote command, useful when do port forward")
	flag.StringVar(&dynamicForward, "D", "", "enable dynamic forward, format [local_host:]local_port")
	flag.Parse()

	if user == "" {
		user = os.Getenv("USER")
	}

	auth := []ssh.AuthMethod{}

	// read ssh agent and default auth key
	if pass == "" && key == "" {
		agentEnv := os.Getenv("SSH_AUTH_SOCK")
		if agentEnv != "" {
			if sock, err := net.Dial("unix", agentEnv); err == nil {
				ag := agent.NewClient(sock)
				if signers, err := ag.Signers(); err == nil {
					//log.Printf("add agent...")
					auth = append(auth, ssh.PublicKeys(signers...))
				}
			}
		}

		home := os.Getenv("HOME")
		for _, f := range []string{
			".ssh/id_rsa",
			".ssh/id_dsa",
			".ssh/identity",
			".ssh/id_ecdsa",
			".ssh/id_ed25519",
		} {
			k1 := filepath.Join(home, f)
			if _, err := os.Stat(k1); err == nil {
				if pemBytes, err := ioutil.ReadFile(k1); err == nil {
					if priKey, err := ssh.ParsePrivateKey(pemBytes); err == nil {
						//log.Printf("add pri...")
						auth = append(auth, ssh.PublicKeys(priKey))
					}
				}
			}
		}
	}

	args := flag.Args()
	var cmd string
	switch len(args) {
	case 0:
		log.Fatal("you must specify the remote host")
	case 1:
		host = args[0]
		cmd = ""
	default:
		host = args[0]
		cmd = strings.Join(args[1:], " ")
	}

	if strings.Contains(host, "@") {
		ss := strings.SplitN(host, "@", 2)
		user = ss[0]
		host = ss[1]
	}

	if pass != "" {
		auth = append(auth, ssh.Password(pass))
	}

	if key != "" {
		pemBytes, err := ioutil.ReadFile(key)
		if err != nil {
			log.Fatal(err)
		}
		priKey, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			log.Fatal(err)
		}
		auth = append(auth, ssh.PublicKeys(priKey))
	}

	exitCh = make(chan int, 5)
	//doneCh = make(chan int, 5)

	config := &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 10 * time.Second,
	}

	if port == "0" {
		port = "22"
	}

	h := net.JoinHostPort(host, port)
	client, err := sshDialTimeout("tcp", h, config, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		log.Fatal(err)
	}

	if localForward != "" {
		waitGroup.Add(1)
		go handleLocalForward(localForward, client)
	}

	if remoteForward != "" {
		waitGroup.Add(1)
		go handleRemoteForward(remoteForward, client)
	}

	if dynamicForward != "" {
		waitGroup.Add(1)
		go handleDynamicForward(dynamicForward, client)
	}

	// start shell
	if cmd == "" && !notRunCmd {
		session.Stdin = os.Stdin
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		// this make CTRL+C works
		oldState, _ := terminal.MakeRaw(0)

		w, h, _ := terminal.GetSize(0)
		if err := session.RequestPty("xterm", h, w, modes); err != nil {
			terminal.Restore(0, oldState)
			log.Fatal(err)
		}
		if err := session.Shell(); err != nil {
			terminal.Restore(0, oldState)
			log.Fatal(err)
		}

		session.Wait()
		terminal.Restore(0, oldState)

		//wait port forward
		if localForward != "" || remoteForward != "" || dynamicForward != "" {
			go registerSignal()

			waitGroup.Wait()
			log.Printf("done.\r\n")
		}
		return
	}

	if !notRunCmd {
		// run command
		output, err := session.CombinedOutput(cmd)
		if len(output) != 0 {
			os.Stdout.Write(output)
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	//wait port forward
	if localForward != "" || remoteForward != "" || dynamicForward != "" {
		go registerSignal()
		waitGroup.Wait()
		log.Printf("done.\r\n")
	}

}

func registerSignal() {
	c := make(chan os.Signal, 5)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-c:
		log.Printf("os signal recevied, signal exit...\r\n")
		for i := 0; i < 5; i++ {
			select {
			case exitCh <- 1:
			default:
			}
		}
	}
}
