package main

import (
	"fmt"
	"testing"
)

func TestChecknet(t *testing.T) {
	fmt.Printf("%#v\n", checknet("https://www.simicloud.com/media/httpbin/ip"))
	fmt.Printf("%#v\n", checknet("http://www.simicloud.com/media/httpbin/status/404"))
}
