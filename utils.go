package main

import (
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
