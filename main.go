package main

import (
	"flag"
)

func main() {
	listenPort := flag.Int("listen", -1, "port to listen on")
	connectPort := flag.Int("connect", -1, "port to connect to")
	createAndMine := flag.Bool("create", false, "create and mine a post, as a test")
	flag.Parse()
	if *connectPort != -1 {
		connect(*connectPort)
	}
	if *createAndMine {
		go func() {
			post := Post{
				payloadRaw: []byte{5, 0, 2, 1},
			}
			post.insert()
			post.mine(20000000)

		}()
	}
	listen(*listenPort) //this goes last because it blocks
}
