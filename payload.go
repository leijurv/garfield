package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

type PayloadHash [32]byte
type Payload []byte

func (hash PayloadHash) Sentiment(nonce Nonce) (bool, [32]byte) {
	positive := hash.Work(nonce, true)
	negative := hash.Work(nonce, false)
	if bytes.Compare(positive[:], negative[:]) < 0 { //return true if positive is less than negative
		Debug.Println("Choosing", positive[:], "over", negative[:])
		return true, positive
	} else {
		Debug.Println("Choosing", negative[:], "over", positive[:])
		return false, negative
	}
}
func (hash PayloadHash) Work(nonce Nonce, positiveSentiment bool) [32]byte { // hash(hash(hash(payolad)+"up")+nonce)
	var sent []byte
	if positiveSentiment {
		sent = []byte("5021")
	} else {
		sent = []byte("1738")
	}
	tmp := sha256.Sum256(append(hash[:], sent...))
	res := sha256.Sum256(append(tmp[:], nonce[:]...))
	return res
}
func (payload Payload) BodyHash() [32]byte {
	return sha256.Sum256(payload)
}
