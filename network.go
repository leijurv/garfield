package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
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
	nonce, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	meta, err := readMeta(peer)
	if err != nil {
		return err
	}
	tieneBytes := make([]byte, 1)
	_, err = io.ReadFull(peer.Conn, tieneBytes)
	if err != nil {
		return err
	}
	tiene := tieneBytes[0]
	onNonceUpdateReceived(payloadHash, *meta, nonce, tiene, peer)
	return nil
}
func readPacketPayloadRequest(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	onPayloadRequested(payloadHash, peer)
	return nil
}

func readPacketPayload(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	meta, err := readMeta(peer)
	if err != nil {
		return err
	}
	payloadBodyHash, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	chk := sha256.Sum256(append(payloadBodyHash[:], meta.raw...))
	if !bytes.Equal(chk[:], payloadHash[:]) {
		fmt.Println("THEY ARE DIFFERENT", chk[:], payloadHash[:], payloadBodyHash[:], meta.raw)
		return errors.New("BADBADBAD LIAR LIAR PANTS ON FIRE")
	}
	payloadLenBytes := make([]byte, 2)
	_, err = io.ReadFull(peer.Conn, payloadLenBytes)
	if err != nil {
		return err
	}
	payloadLen := int(binary.LittleEndian.Uint16(payloadLenBytes))
	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(peer.Conn, payload)
	if err != nil {
		return err
	}
	onPayloadReceived(payloadHash, *meta, payloadBodyHash, payload)
	return nil
}
func readPacketGetNonce(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	onGetNonce(payloadHash, peer)
	return nil
}
func readPacketMultiNonce(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return err
	}
	meta, err := readMeta(peer)
	if err != nil {
		return err
	}
	nonceCountBytes := make([]byte, 2)
	_, err = io.ReadFull(peer.Conn, nonceCountBytes)
	if err != nil {
		return err
	}
	nonceCount := int(binary.LittleEndian.Uint16(nonceCountBytes))
	nonces := make([]Nonce, nonceCount)
	for i := 0; i < nonceCount; i++ {
		nonce, err := read32(peer.Conn)
		if err != nil {
			return err
		}
		nonces[i] = nonce
	}
	onPacketMultiNonce(payloadHash, nonces, *meta, peer)
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
		go AddPeer(conn)
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
