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
	"sync"
	"time"
)

var timeWait = 10

var countTotal = map[string]int{}
var countLock = &sync.Mutex{}

func check(u string, max int, errch chan int) {
	log.Printf("checking %s...", u)
	//var count int
	for {
		time.Sleep(time.Duration(timeWait) * time.Second)
		if dialing {
			continue
		}

		ret := checknet(u)

		// check again
		if dialing {
			continue
		}

		countLock.Lock()
		if ret {
			countTotal[u] = 0
			//count = 0
		} else {
			//count++
			countTotal[u] = countTotal[u] + 1
		}

		if countTotal[u] >= max {
			errch <- 1
			//count = 0
		}

		countLock.Unlock()
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
		log.Printf("url %s check error", u)
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
	flag.IntVar(&busyWait, "busy_wait", 30, "seconds to wait when network busy")
	flag.IntVar(&dialWait, "dial_wait", 20, "seconds to wait during re-dial")
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

	countLock.Lock()
	for _, dn := range domains {
		countTotal[dn] = 0
		go check(dn, maxErr, errch)
	}
	countLock.Unlock()

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
			countLock.Lock()
			for k := range countTotal {
				countTotal[k] = 0
			}
			countLock.Unlock()
			dialing = false
		case <-time.After(30 * time.Minute):
		}
	}
}
