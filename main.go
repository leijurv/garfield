package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	// debugOut redirects debug logs to os.Stdout
	debugOut = os.Stdout
	// infoOut redirects info logs to os.Stdout
	infoOut = os.Stdout
	// warningOut redirects warning logs to os.Stdout
	warningOut = os.Stdout
	// errorOut redirects error logs to os.Stderr
	errorOut = os.Stderr
)

type IntSliceFlag []string

func (i *IntSliceFlag) String() string {
	return fmt.Sprintf("%d", *i)
}

func (i *IntSliceFlag) Set(value string) error {

	*i = append(*i, value)
	return nil
}

func main() {
	var listenPort int
	var connectPorts IntSliceFlag
	var createAndMine bool

	flag.IntVar(&listenPort, "listen", 0, "port to listen on")
	flag.Var(&connectPorts, "connect", "ports to connect to")
	flag.BoolVar(&createAndMine, "create", false, "create and mine a post, as a test")

	flag.Parse()

	if len(connectPorts) > 0 {
		for _, port := range connectPorts {
			err := Connect(port)
			if err != nil {
				Debug.Printf("Couldn't connect to peer on port: %v. Error: %v", port, err)
			}
		}

	}
	/*if createAndMine {
		go func() {
			post := Post{
				PayloadRaw: []byte{5, 0, 2, 1},
			}
			post.Insert()
			post.Mine(20000000)

		}()
	}*/

	// This goes last because it blocks
	err := Listen(listenPort)
	if err != nil {
		panic(err)
	}
}
