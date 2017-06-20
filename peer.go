package main

import (
	"fmt"
	"io"
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
	panic("Peer wasn't in list to begin with")
}

// AddPeer appends a peer to the peer list
func AddPeer(conn net.Conn) {
	peer := Peer{Conn: conn}
	peersLock.Lock()
	defer peersLock.Unlock()
	peers = append(peers, &peer)

	go func() {
		err := peer.Listen()
		fmt.Println("Disconnected beacuse of", err)
	}()
}
func Broadcast(data []byte) {
	peersLock.Lock()
	defer peersLock.Unlock()
	for i := 0; i < len(peers); i++ {
		peer := peers[i]
		go peer.Send(data)
	}
}

// Listen starts a listener for notifications to that peer
func (peer *Peer) Listen() error {
	defer peer.Conn.Close()
	defer peer.remove()
	msgType := make([]byte, 1)
	for {
		_, err := io.ReadFull(peer.Conn, msgType)
		if err != nil {
			return err
		}
		packetType := PacketType(msgType[0])
		handle, ok := packetHandlers[packetType]
		if !ok || handle == nil {
			return fmt.Errorf("Unexpected prefix byte %d", int(msgType[0]))
		}
		err = handle(peer)
		if err != nil {
			return err
		}

	}
}
