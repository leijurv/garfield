package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

func readPacketNonceUpdate(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	newNonce, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	fmt.Println("Someone gave us a new nonce", newNonce, "for", payloadHash)
	go onNonceUpdateReceived(payloadHash, newNonce, peer)
	return nil
}
func readPacketPostContentsRequest(peer *Peer) error {
	requestedPayloadHash, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	fmt.Println("Someone just asked us for post contents for payload hash", requestedPayloadHash)
	go onPostContentsRequested(requestedPayloadHash, peer)
	return nil
}
func readPacketPostContents(peer *Peer) error {
	payloadLenBytes := make([]byte, 2)
	_, err := io.ReadFull(peer.Conn, payloadLenBytes)
	if err != nil {
		return err
	}
	payloadLen := int(binary.LittleEndian.Uint16(payloadLenBytes))
	fmt.Println("Reading payload with len", payloadLen)
	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(peer.Conn, payload)
	if err != nil {
		return err
	}
	nonce, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	fmt.Println("Someone gave us post contents")
	go onPostContentsReceived(payload, nonce, peer)
	return nil
}

// Listen is the listener port to get notifications
func Listen(port int) error {
	listen, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	fmt.Println("Listening on", port)
	for {
		conn, err := listen.Accept()
		if err != nil {
			return err
		}
		fmt.Println("Connection from ", conn)
		AddPeer(conn)
	}
}

// Connect connects and adds a peer
func Connect(port string) error {
	fmt.Println("Connecting to", port)
	conn, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	AddPeer(conn)
	return nil
}
