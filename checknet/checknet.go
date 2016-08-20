package main

import (
	//"net"
	//"net/http"
	"fmt"
	"log"
	//"os"
	//"bytes"
	"flag"
	"os/exec"
	"strings"
	"time"
)

var timeWait = 10

func check(u string, max int, errch chan int) {
	log.Printf("checking %s...", u)
	var count int
	for {
		time.Sleep(time.Duration(timeWait) * time.Second)
		if dialing {
			count = 0
			continue
		}
		if checknet(u) {
			count = 0
		} else {
			count++
		}

		if count >= max {
			errch <- 1
			count = 0
		}
	}
}

func checknet(u string) bool {
	args := fmt.Sprintf("-o /dev/null --connect-timeout %d --speed-limit %d --speed-time %d -s -S -k -L %s",
		3, 10*1024, 5,
		u,
	)
	cmd := exec.Command("curl", strings.Fields(args)...)
	out, err := cmd.CombinedOutput()
	if len(out) != 0 {
		log.Printf("%s", string(out))
	}
	if err != nil {
		return false
	}

	return true
}

func redial(c chan int) {
	log.Println("redialing...")

	cmd := exec.Command("sudo", "poff", "dsl-provider")
	out, err := cmd.CombinedOutput()
	if len(out) != 0 {
		log.Printf("%s", string(out))
	}
	if err != nil {
		log.Printf("%s", err.Error())
	}

	time.Sleep(time.Duration(dialWait) * time.Second)
	cmd = exec.Command("sudo", "pon", "dsl-provider")

	out, err = cmd.CombinedOutput()
	if len(out) != 0 {
		log.Printf("%s", string(out))
	}
	if err != nil {
		log.Printf("%s", err.Error())
	}

	time.Sleep(time.Duration(dialWait) * time.Second)
	c <- 1
	log.Println("done")
}

var dialing = false
var maxErr = 3
var idleWait, busyWait, dialWait int

func main() {
	flag.IntVar(&maxErr, "max_error", 3, "max error on check to trigger re-dial")
	flag.IntVar(&idleWait, "idle_wait", 60, "seconds to wait when network not busy")
	flag.IntVar(&busyWait, "busy_wait", 10, "seconds to wait when network busy")
	flag.IntVar(&dialWait, "dial_wait", 5, "seconds to wait during re-dial")
	flag.Parse()

	domains := []string{
		//"https://www.ratafee.nl/stream-bytes/10240",
		"https://www.httpbin.org/ip",
		"https://www.baidu.com/",
		"http://www.ip.cn/",
		"https://www.taobao.com/",
	}
	args := flag.Args()
	if len(args) > 0 {
		domains = args
	}

	errch := make(chan int, len(domains))

	for _, dn := range domains {
		go check(dn, maxErr, errch)
	}

	dialDone := make(chan int)

	for {
		h := time.Now().Hour()
		if h >= 19 && h < 24 {
			timeWait = busyWait
		} else {
			timeWait = idleWait
		}

		select {
		case <-errch:
			if !dialing {
				dialing = true
				go redial(dialDone)
			}
		case <-dialDone:
			dialing = false
		case <-time.After(30 * time.Minute):
		}
	}
}
