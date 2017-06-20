package main

import (
	"encoding/json"
	"io"
	"math/rand"
	"time"
)

func read32(conn io.Reader) ([32]byte, error) {
	var data [32]byte
	_, err := io.ReadFull(conn, data[:])
	return data, err
}

func randomNonce() [32]byte {
	nr := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	fixedLengthIsBS := make([]byte, 32)
	nr.Read(fixedLengthIsBS)

	var nonce [32]byte
	copy(nonce[:], fixedLengthIsBS)
	return nonce
}

func readMeta(peer *Peer) (*Meta, error) {
	metaLenBytes := make([]byte, 1)
	_, err := io.ReadFull(peer.Conn, metaLenBytes)
	if err != nil {
		return nil, err
	}
	metaLen := int(uint8(metaLenBytes[0]))

	metaBytes := make([]byte, metaLen)
	_, err = io.ReadFull(peer.Conn, metaBytes)
	if err != nil {
		return nil, err
	}
	meta := Meta{raw: metaBytes}
	err = json.Unmarshal(metaBytes, &(meta.Data))
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
