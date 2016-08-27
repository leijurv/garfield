package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

func (peer *Peer) listen() error {
	for {
		msgType := make([]byte, 1)
		_, err := peer.conn.Read(msgType)
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
			peer.conn.Close()
			fmt.Println("Unexpected prefix byte", msgType)
		}
	}
}
func readPacketNonceUpdate(peer *Peer) error {
	payloadHash, err := read32(peer.conn)
	if err != nil {
		return err
	}
	newNonce, err := read32(peer.conn)
	if err != nil {
		return err
	}
	fmt.Println("Someone gave us a new nonce", newNonce, "for", payloadHash)
	go onNonceUpdateReceived(payloadHash, newNonce, peer)
	return nil
}
func readPacketPostContentsRequest(peer *Peer) error {
	requestedPayloadHash, err := read32(peer.conn)
	if err != nil {
		return err
	}
	fmt.Println("Someone just asked us for post contents for payload hash", requestedPayloadHash)
	go onPostContentsRequested(requestedPayloadHash, peer)
	return nil
}
func readPacketPostContents(peer *Peer) error {
	payloadLenBytes := make([]byte, 2)
	_, err := io.ReadFull(peer.conn, payloadLenBytes)
	if err != nil {
		return err
	}
	payloadLen := int(binary.LittleEndian.Uint16(payloadLenBytes))
	fmt.Println("Reading payload with len", payloadLen)
	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(peer.conn, payload)
	if err != nil {
		return err
	}
	nonce, err := read32(peer.conn)
	if err != nil {
		return err
	}
	fmt.Println("Someone gave us post contents")
	go onPostContentsReceived(payload, nonce, peer)
	return nil
}
func listen(port int) {
	listen, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on", port)
	for {
		conn, err := listen.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("Connection from ", conn)
		addPeer(conn)
	}
}
func connect(port int) {
	fmt.Println("Connecting to", port)
	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	addPeer(conn)
}
