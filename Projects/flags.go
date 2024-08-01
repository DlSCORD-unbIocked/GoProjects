package main

import (
	"flag"
	"fmt"
)

func main() {
	test := flag.String("test", "Test", "testing")
	flag.Parse()
	fmt.Println(*test)
}
