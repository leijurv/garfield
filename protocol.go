package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	PacketNonceUpdate         uint8 = 4
	PacketPostContents        uint8 = 7
	PacketPostContentsRequest uint8 = 9
)

func onNonceUpdateReceived(postPayloadHash [32]byte, newNonce [32]byte, peerFrom *Peer) {
	postsLock.Lock()
	post, ok := posts[postPayloadHash]

	if ok {
		comparison := post.checkPossibleNonce(newNonce)
		if comparison < 0 {
			oldNonce := post.nonce
			post.nonce = newNonce
			post.mostRecentNonceUpdate = time.Now()
			postsLock.Unlock()
			fmt.Println("Updating post nonce from", oldNonce[:], "to", newNonce[:], ". Post hash is now", post.hash())
			post.broadcastNonceUpdate()
		} else {
			postsLock.Unlock()
			if comparison != 0 { //its not equal, they actaully just proudly gave us a nonce that's WORSE
				//wow you're behind the times. let's help you out
				fmt.Println("Helping out peer", peerFrom, " that's behind the times")
				go peerFrom.send(append(append([]byte{PacketNonceUpdate}, postPayloadHash[:]...), post.nonce[:]...))
			}
		}
	} else {
		postsLock.Unlock()
		//ask the peer we received this from for the contents of the post because we don't have it
		peerFrom.send(append([]byte{PacketPostContentsRequest}, postPayloadHash[:]...))
	}

}

func onPostContentsReceived(payloadRaw []byte, nonce [32]byte, peerFrom *Peer) {
	fmt.Println("Post contents:", payloadRaw)
	payloadHash := sha256.Sum256(payloadRaw)
	postsLock.Lock()
	_, ok := posts[payloadHash]
	if ok {
		fmt.Println("Already have it")
		postsLock.Unlock()
		//already have it. let's just check if the nonce is better
		onNonceUpdateReceived(payloadHash, nonce, peerFrom)
	} else {
		//dont have it, lets add it
		now := time.Now()
		post := Post{
			payloadRaw:            payloadRaw,
			nonce:                 nonce,
			firstReceived:         now,
			mostRecentNonceUpdate: now,
		}
		posts[payloadHash] = &post
		postsLock.Unlock()
		fmt.Println("Added post with payload hash", post.payloadHash(), "and normal hash", post.hash())
	}
}
func onPostContentsRequested(payloadHash [32]byte, peerFrom *Peer) {
	postsLock.Lock()
	post, ok := posts[payloadHash]
	postsLock.Unlock()
	if ok {
		payloadLenBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(payloadLenBytes, uint16(len(post.payloadRaw)))
		fmt.Println("Sending contents of ", payloadHash)
		//man I wish go was better at appending mulitple arrays. lol im probbaly doing something wrong here. BUT HEY, IT WORKS
		message := append(append(append([]byte{PacketPostContents}, payloadLenBytes...), post.payloadRaw...), post.nonce[:]...)
		fmt.Println("data:", message)
		peerFrom.send(message)
	} else {
		//um idk we don't have it. just ignore lol
	}
}
