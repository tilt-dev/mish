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
		handleErr("error setting up", err)
	}

	err = sh.Run()
	if err != nil {
		handleErr("error running", err)
	}
}

func handleErr(msg string, err error) {
	// print to stderr so user can see it w/o digging in logfile
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	log.Fatalf("%s: %v", msg, err)
}
