package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yisrael-haber/go-http-server/src"
)

func main() {

	if len(os.Args) > 1 && os.Args[1] == "help" {
		src.DisplayHelp()
		return
	}

	port, loc, err := src.ExtractArgs(os.Args[1:])

	if err != nil {
		fmt.Printf("Encountered error while setting port and server location: \n%s", err.Error())
		os.Exit(1)
	}

	log.Printf("Preparing to serve location %s", loc)
	log.Printf("Binding to localhost on port %d\n", port)

	err = os.Chdir(loc)
	if err != nil {
		fmt.Printf("Encountered error while changing working directory:\n\t%s\n", err.Error())
	}

	listener := src.BindPort(port)
	defer listener.Close()

	src.AcceptAndHandleConnections(listener)
}
