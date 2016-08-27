package main

import (
	"fmt"
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
// AddPeer appends a peer to the peer list
func AddPeer(conn net.Conn) {
	peer := Peer{Conn: conn}
	peersLock.Lock()
	peers = append(peers, &peer)
	peersLock.Unlock()
	go peer.Listen()
}

// Listen starts a listener for notifications to that peer
func (peer *Peer) Listen() error {
	defer peer.Conn.Close()
	defer peer.remove()
	for {
		msgType := make([]byte, 1)
		_, err := peer.Conn.Read(msgType)
		if err != nil {
			return err
		}
		switch msgType[0] {
		case PacketNonceUpdate:
			err := readPacketNonceUpdate(peer)
			if err != nil {
				return err
			}
		case PacketPostContents:
			err := readPacketPostContents(peer)
			if err != nil {
				return err
			}
		case PacketPostContentsRequest:
			err := readPacketPostContentsRequest(peer)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unexpected prefix byte %d", int(msgType[0]))
		}
	}
}
