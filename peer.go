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
	writeLock sync.Mutex //not a rw because it's only for outgoing
}

var peers []*Peer
var peersLock sync.RWMutex //rw because the most common operation will be broadcast, which can happen concurrently, and w operations (adding/removing) will be (hopefully) uncommon

// Send writes a message to the conn
func (peer *Peer) Send(msg []byte) error {
	peer.writeLock.Lock()
	defer peer.writeLock.Unlock()
	_, err := peer.Conn.Write(msg)
	return err
}
func (peer *Peer) remove() {
	peersLock.Lock() //write lock because we are removing
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
	peersLock.Lock() //write lock because we are appending
	defer peersLock.Unlock()
	peers = append(peers, &peer)

	go func() {
		err := peer.Listen() //note that once peer.Listen returns, it calls peer.remove and peer.Conn.Close in a defer, so no need to do that here
		fmt.Println("Disconnected", conn, "beacuse of", err)
	}()
}
func Broadcast(data []byte) { //this does not return an error because it's a best-effort broadcast, it'll just try as many as it can in new goroutines and not bother the caller about any possible sending errors
	peersLock.RLock() //rlock because we are only reading the list, and just starting a goroutine for each, so this should make broadcasting very very fast and concurrent
	defer peersLock.RUnlock()
	for i := 0; i < len(peers); i++ {
		peer := peers[i]
		go peer.Send(data) //TODO, disconnect and remove on error?
	}
}

// Listen starts a listener for notifications to that peer
func (peer *Peer) Listen() error {
	defer peer.Conn.Close()
	defer peer.remove()
	for {
		packetTypeBytes := make([]byte, 1)
		_, err := io.ReadFull(peer.Conn, packetTypeBytes)
		if err != nil {
			return err
		}
		packetType := PacketType(packetTypeBytes[0])
		packetHandler, ok := packetHandlers[packetType]
		if !ok || packetHandler == nil {
			return fmt.Errorf("Unexpected prefix byte %d", int(packetType))
		}
		err = packetHandler(peer)
		if err != nil {
			return err
		}

	}
}
