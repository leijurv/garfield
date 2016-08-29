package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

// Update ids sent to peers
const (
	PacketNonceUpdate         uint8 = iota
	PacketPostContents
	PacketPostContentsRequest
)

func onNonceUpdateReceived(postPayloadHash [32]byte, newNonce [32]byte, peerFrom *Peer) error {
	postsLock.Lock()
	post, ok := posts[postPayloadHash]

	if ok {
		comparison := post.CheckPossibleNonce(newNonce)
		if comparison < 0 {
			oldNonce := post.nonce
			post.updateNonce(newNonce)
			postsLock.Unlock()
			fmt.Println("Updating post nonce from", oldNonce[:], "to", newNonce[:], ". Post hash is now", post.Hash())
			post.BroadcastNonceUpdate()
		} else {
			postsLock.Unlock()
			if comparison != 0 { //its not equal, they actaully just proudly gave us a nonce that's WORSE
				//wow you're behind the times. let's help you out
				fmt.Println("Helping out peer", peerFrom, " that's behind the times")
				err := peerFrom.Send(append(append([]byte{PacketNonceUpdate}, postPayloadHash[:]...), post.nonce[:]...))
				if err != nil {
					return err
				}
			}
		}
	} else {
		postsLock.Unlock()
		//ask the peer we received this from for the contents of the post because we don't have it
		err :=peerFrom.Send(append([]byte{PacketPostContentsRequest}, postPayloadHash[:]...))
		if err != nil {
			return err
		}
	}
	return nil
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
			PayloadRaw:            payloadRaw,
			nonce:                 nonce,
			firstReceived:         now,
			mostRecentNonceUpdate: now,
		}
		posts[payloadHash] = &post
		postsLock.Unlock()
		fmt.Println("Added post with payload hash", post.PayloadHash(), "and normal hash", post.Hash())
	}
}
func onPostContentsRequested(payloadHash [32]byte, peerFrom *Peer) error {
	postsLock.Lock()
	post, ok := posts[payloadHash]
	postsLock.Unlock()
	if ok {
		payloadLenBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(payloadLenBytes, uint16(len(post.PayloadRaw)))
		fmt.Println("Sending contents of ", payloadHash)
		//man I wish go was better at appending mulitple arrays. lol im probbaly doing something wrong here. BUT HEY, IT WORKS
		message := append(append(append([]byte{PacketPostContents}, payloadLenBytes...), post.PayloadRaw...), post.nonce[:]...)
		fmt.Println("data:", message)
		err := peerFrom.Send(message)
		if err != nil {
			return err
		}
	}
	//um idk we don't have it. just ignore lol

	return nil
}
