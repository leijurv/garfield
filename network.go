package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
)

type PeerRemovalError struct {
	error
}

func (err *PeerRemovalError) IsPeerRemovalError() bool {
	return true
}

var (
	ErrMismatchedShasums PeerRemovalError = PeerRemovalError{errors.New("peer: shasums do not match")}
)

func readPacketNonceUpdate(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return PeerRemovalError{err}
	}
	nonce, err := read32(peer.Conn)
	if err != nil {
		return PeerRemovalError{err}
	}
	meta, err := readMeta(peer)
	if err != nil {
		return PeerRemovalError{err}
	}
	tieneBytes := make([]byte, 1)
	_, err = io.ReadFull(peer.Conn, tieneBytes)
	if err != nil {
		return PeerRemovalError{err}
	}
	tiene := tieneBytes[0]
	return onNonceUpdateReceived(payloadHash, *meta, nonce, tiene, peer)
}
func readPacketPayloadRequest(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return PeerRemovalError{err}
	}
	return onPayloadRequested(payloadHash, peer)
}
func readPacketPayload(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return PeerRemovalError{err}
	}
	meta, err := readMeta(peer)
	if err != nil {
		return PeerRemovalError{err}
	}
	payloadBodyHash, err := read32(peer.Conn)
	if err != nil {
		return PeerRemovalError{err}
	}
	chk := sha256.Sum256(append(payloadBodyHash[:], meta.raw...))
	if !bytes.Equal(chk[:], payloadHash[:]) {
		Warning.Println("Shasums do not match: ", chk[:], payloadHash[:], payloadBodyHash[:], meta.raw)
		return ErrMismatchedShasums
	}
	payloadLenBytes := make([]byte, 2)
	_, err = io.ReadFull(peer.Conn, payloadLenBytes)
	if err != nil {
		return PeerRemovalError{err}
	}
	payloadLen := int(binary.LittleEndian.Uint16(payloadLenBytes))
	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(peer.Conn, payload)
	if err != nil {
		return PeerRemovalError{err}
	}
	return onPayloadReceived(payloadHash, *meta, payloadBodyHash, payload)
}

func readPacketGetNonce(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return PeerRemovalError{err}
	}
	return onGetNonce(payloadHash, peer)
}

func readPacketMultiNonce(peer *Peer) error {
	payloadHash, err := read32(peer.Conn)
	if err != nil {
		return PeerRemovalError{err}
	}
	meta, err := readMeta(peer)
	if err != nil {
		return PeerRemovalError{err}
	}
	nonceCountBytes := make([]byte, 2)
	_, err = io.ReadFull(peer.Conn, nonceCountBytes)
	if err != nil {
		return PeerRemovalError{err}
	}
	nonceCount := int(binary.LittleEndian.Uint16(nonceCountBytes))
	nonces := make([]Nonce, nonceCount)
	for i := 0; i < nonceCount; i++ {
		nonce, err := read32(peer.Conn)
		if err != nil {
			return PeerRemovalError{err}
		}
		nonces[i] = nonce
	}
	return onPacketMultiNonce(payloadHash, nonces, *meta, peer)
}

// Listen is the listener port to get notifications
func Listen(port int) error {
	listen, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	Info.Println("Listening on", port)
	for {
		conn, err := listen.Accept()
		if err != nil {
			return err
		}
		Debug.Println("Connection from ", conn)
		go AddPeer(conn)
	}
}

// Connect connects and adds a peer
func Connect(port string) error {
	Debug.Println("Connecting to", port)
	conn, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	AddPeer(conn)
	return nil
}

func IsPeerRemovalErr(err error) bool {
	_, ok := err.(PeerRemovalError)
	return ok
}
