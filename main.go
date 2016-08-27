package main

import (
	"flag"
)

func main() {
	var listenPort int
	var connectPort int
	var createAndMine bool

	flag.IntVar(&listenPort, "listen", 0, "port to listen on")
	flag.IntVar(&connectPort, "connect", 0, "port to connect to")
	flag.BoolVar(&createAndMine, "create", false, "create and mine a post, as a test")

	flag.Parse()

	if connectPort > 0 {
		err := Connect(connectPort)
		if err != nil {
			panic(err)
		}
	}
	if createAndMine {
		go func() {
			post := Post{
				PayloadRaw: []byte{5, 0, 2, 1},
			}
			post.Insert()
			post.Mine(20000000)

		}()
	}

	// This goes last because it blocks
	err := Listen(listenPort)
	if err != nil {
		panic(err)
	}
}
