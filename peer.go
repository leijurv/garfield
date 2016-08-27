package main

import (
	"net"
	"sync"
)

// Peer is the managed structure for all connected nodes
type Peer struct {
	conn      net.Conn
	writeLock sync.Mutex
}

var peers []*Peer
var peersLock sync.Mutex

func (peer *Peer) Send(msg []byte) error {
	peer.writeLock.Lock()
	defer peer.writeLock.Unlock()
	_, err := peer.conn.Write(msg)
	return err
}
func AddPeer(conn net.Conn) {
	peer := Peer{conn: conn}
	peersLock.Lock()
	peers = append(peers, &peer)
	peersLock.Unlock()
	go peer.Listen()
}
