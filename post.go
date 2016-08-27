package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// Post is a post to garfield
type Post struct {
	PayloadRaw            []byte
	nonce                 [32]byte
	firstReceived         time.Time
	mostRecentNonceUpdate time.Time
}

var posts = make(map[[32]byte]*Post)
var postsLock sync.Mutex

// PayloadHash gets the hash of the payload
func (post *Post) PayloadHash() [32]byte {
	return sha256.Sum256(post.PayloadRaw)
}

// Hash gets the hash of the payload and the nonce
func (post *Post) Hash() [32]byte {
	return post.hashNonce(post.nonce)
}

func (post *Post) hashNonce(nonce [32]byte) [32]byte {
	combined := append(post.PayloadRaw, nonce[:]...)
	return sha256.Sum256(combined)
}

// CheckPossibleNonce compares the new nonce hash with the old one
func (post *Post) CheckPossibleNonce(newNonce [32]byte) int {
	if bytes.Equal(newNonce[:], post.nonce[:]) {
		return 0
	}
	newHash := post.hashNonce(newNonce)
	oldHash := post.Hash()
	comparison := bytes.Compare(newHash[:], oldHash[:])
	return comparison
}

// Insert adds a post to the list
func (post *Post) Insert() {
	postsLock.Lock()
	posts[post.PayloadHash()] = post
	postsLock.Unlock()
}

// Mine does something
func (post *Post) Mine(count int) {
	currentHash := post.Hash()
	nonce := randomNonce()
	for i := 0; i < count; i++ {
		newHash := sha256.Sum256(append(post.PayloadRaw, nonce[:]...))
		if bytes.Compare(newHash[:], currentHash[:]) < 0 {
			currentHash = newHash
			postsLock.Lock()
			post.nonce = nonce
			post.mostRecentNonceUpdate = time.Now()
			postsLock.Unlock()
			fmt.Println("Nonce improvement, hash is now ", newHash)
			post.BroadcastNonceUpdate()
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

// BroadcastNonceUpdate sends out the update to all peers
func (post *Post) BroadcastNonceUpdate() {
	postPayloadHash := post.PayloadHash()
	newNonce := post.nonce
	message := append(append([]byte{PacketNonceUpdate}, postPayloadHash[:]...), newNonce[:]...)
	peersLock.Lock()
	fmt.Println("Sending nonce update")
	for _, peer := range peers {
		go peer.Send(message)
	}
	peersLock.Unlock()
}

// Listen starts a listener for notifications to that peer
func (peer *Peer) Listen() error {
	defer peer.Conn.Close()
	defer peer.remove()
	for {
		msgType := make([]byte, 1)
		_, err := peer.Conn.Read(msgType)
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
			return fmt.Errorf("Unexpected prefix byte %d", int(msgType[0]))
		}
	}
}