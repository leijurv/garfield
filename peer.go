package main

import (
	"net"
	"sync"
)

// Peer is the managed structure for all connected nodes
type Peer struct {
	Conn      net.Conn
	writeLock sync.Mutex
}

var peers []*Peer
var peersLock sync.Mutex

// Send writes a message to the conn
func (peer *Peer) Send(msg []byte) error {
	peer.writeLock.Lock()
	defer peer.writeLock.Unlock()
	_, err := peer.Conn.Write(msg)
	return err
}
// AddPeer appends a peer to the peer list
func AddPeer(conn net.Conn) {
	peer := Peer{Conn: conn}
	peersLock.Lock()
	peers = append(peers, &peer)
	peersLock.Unlock()
	go peer.Listen()
}
