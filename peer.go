package main

import (
	"fmt"
	"net"
	"sync"
)

type Peer struct {
	conn      net.Conn
	writeLock sync.Mutex
}

var peers []*Peer
var peersLock sync.Mutex

func (peer *Peer) send(msg []byte) error {
	peer.writeLock.Lock()
	defer peer.writeLock.Unlock()
	_, err := peer.conn.Write(msg)
	return err
}
func (peer *Peer) remove() {
	peersLock.Lock()
	defer peersLock.Unlock()
	for i := 0; i < len(peers); i++ {
		if peers[i] == peer {
			fmt.Println("Removing peer from list")
			peers = append(peers[:i], peers[i+1:]...)
			return
		}
	}
	fmt.Println("Peer wasn't in list to begin with")
}
func addPeer(conn net.Conn) {
	peer := Peer{conn: conn}
	peersLock.Lock()
	peers = append(peers, &peer)
	peersLock.Unlock()
	go peer.listen()
}
