package main

import (
	"expvar"
	"fmt"
	"log"
	"os"

	"github.com/windmilleng/mish/mish"
)

func main() {
	expvar.NewString("serviceName").Set("mish")
	sh, err := mish.Setup()
	if err != nil {
		log.Fatalf("error setting up: %v", err)
	}

	// here's a comment, ive changed mish code!

	err = sh.Run()
	if err != nil {
		// make sure it's not hidden
		fmt.Fprintf(os.Stderr, "error running: %v\n", err)
		log.Fatalf("error running: %v", err)
	}
}
