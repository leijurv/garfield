package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

type Post struct {
	payloadRaw            []byte
	nonce                 [32]byte
	firstReceived         time.Time
	mostRecentNonceUpdate time.Time
}

var posts = make(map[[32]byte]*Post)
var postsLock sync.Mutex

func (post *Post) payloadHash() [32]byte {
	return sha256.Sum256(post.payloadRaw)
}
func (post *Post) hash() [32]byte {
	combined := append(post.payloadRaw, post.nonce[:]...)
	return sha256.Sum256(combined)
}
func (post *Post) checkPossibleNonce(newNonce [32]byte) int {
	if bytes.Equal(newNonce[:], post.nonce[:]) {
		return 0
	}
	newHash := sha256.Sum256(append(post.payloadRaw, newNonce[:]...))
	oldHash := post.hash()
	comparison := bytes.Compare(newHash[:], oldHash[:])
	return comparison
}
func (post *Post) insert() {
	postsLock.Lock()
	posts[post.payloadHash()] = post
	postsLock.Unlock()
}
func (post *Post) mine(count int) {
	currentHash := post.hash()
	nonce := randomNonce()
	for i := 0; i < count; i++ {
		newHash := sha256.Sum256(append(post.payloadRaw, nonce[:]...))
		if bytes.Compare(newHash[:], currentHash[:]) < 0 {
			currentHash = newHash
			postsLock.Lock()
			post.nonce = nonce
			post.mostRecentNonceUpdate = time.Now()
			postsLock.Unlock()
			fmt.Println("Nonce improvement, hash is now ", newHash)
			post.broadcastNonceUpdate()
		}
		nonce[31]++
		if nonce[31] == 0 {
			nonce[30]++
			if nonce[30] == 0 {
				for i := 29; i >= 0; i-- {
					nonce[i]++
					if nonce[i] != 0 {
						break
					}
				}
			}
		}
	}
}
func (post *Post) broadcastNonceUpdate() {
	postPayloadHash := post.payloadHash()
	newNonce := post.nonce
	message := append(append([]byte{PacketNonceUpdate}, postPayloadHash[:]...), newNonce[:]...)
	peersLock.Lock()
	fmt.Println("Sending nonce update")
	for _, peer := range peers {
		go peer.send(message)
	}
	peersLock.Unlock()
}
