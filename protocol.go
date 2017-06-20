package main

import (
	//"crypto/sha256"
	"encoding/binary"
	"fmt"
	//"time"
)

const (
	FlagHasPayload = 1 << iota
	FlagHasNonces
	FlagAcceptableScore
	FlagAcceptableMeta
)

func onNonceUpdateReceived(postPayloadHash PayloadHash, meta Meta, newNonce Nonce, tiene byte, peer *Peer) error {
	message := []byte{uint8(PacketNonceUpdate)}
	message = append(message, postPayloadHash[:]...)
	message = append(message, newNonce[:]...)

	if len(meta.raw) > 255 {
		panic("not long enough")
	}
	message = append(message, uint8(len(meta.raw)))
	message = append(message, meta.raw...)
	if !meta.Verify() {
		//we don't care
		if tiene&FlagHasNonces == 0 {
			//we just received from someone who also doesn't know what they are doing
			//if we rebroadcast, it will go back to them with the same flags, and we will have an infinite loop
			//so,
		}
		var flags uint8
		post := GetPost(postPayloadHash)
		if post != nil && post.HasPayload() {
			//don't set HasNonces because we aren't keeping track of the nonces so it would be deceiving
			flags |= FlagHasPayload
		}
		message = append(message, flags)
		Broadcast(message)
		return nil
	}
	post := GetPost(postPayloadHash)
	if post != nil {
		//If we already have either the nonces or the body, let it know we have a new nonce to consider
		if post.insertIfImprovement(newNonce) {
			var flags uint8
			flags |= FlagHasNonces
			flags |= FlagAcceptableMeta
			if post.HasPayload() {
				flags |= FlagHasPayload
			}
			if post.AcceptableScore() {
				flags |= FlagAcceptableScore
			}
			//TODO if the score has only just now become acceptable, go ahead and request that payload
			message = append(message, flags)
			Broadcast(message)
			return nil
		} else {
			//this was not an improvement. so, useless
		}
	} else {
		//TODO honestly maybe just broadcast a getnonce to EVERY peer...

		//TODO rebroadcast this improvement with proper flags here
		if tiene&FlagHasNonces != 0 {
			//ask the peer we received this from for the nonces for the post because we don't have it
			message := []byte{uint8(PacketGetNonce)}
			message = append(message, postPayloadHash[:]...)
			fmt.Println("data:", message)
			err := peer.Send(message)
			if err != nil {
				return err
			}
		} else {
			//this peer was just relaying, they don't have the nonces
			//what a terrible thing to do
			if tiene&FlagAcceptableMeta != 0 {
				//this peer cares. they like the meta.
				//we can't check if they like the score, because they don't have the nonces

				//why would they like the meta but not have the nonces?
				//because they received it through someone who doesn't care
				//in that scenario, they just asked all of their peers if they have the nonces

				//TODO wait 60 seconds then ask them again for the nonces. muahahahhahha
			} else {
				//well. we like the meta. we don't have the nonces. we received from someone who doesn't have the nonces and actually doesn't care about the post at all.
				//TODO broadcast a getnonce to everyone lol
			}
		}

	}

	return nil
}

func onPayloadReceived(payloadHash PayloadHash, meta Meta, payloadBodyHash [32]byte, payloadBody Payload) {
	fmt.Println("Post contents:", payloadBody)
	post := GetPost(payloadHash)
	if post != nil {
		fmt.Println("Cool, a payload")
		post.payloadReceived(payloadBody)
	} else {
		fmt.Println("While appreciated, I did not ask for contents of", payloadHash, "so I don't have its nonces so I can't accept it")
	}
}
func onPayloadRequested(payloadHash PayloadHash, peerFrom *Peer) error {
	//TODO pow
	post := GetPost(payloadHash)
	if post == nil {
		return nil //um idk we don't have it. just ignore lol
	}
	if !post.HasPayload() {
		return nil
	}

	fmt.Println("Sending contents of ", payloadHash)
	//man I wish go was better at appending mulitple arrays. lol im probbaly doing something wrong here. BUT HEY, IT WORKS

	message := []byte{uint8(PacketPayload)}
	message = append(message, payloadHash[:]...)
	if len(post.Meta.raw) > 255 {
		panic("not long enough")
	}
	message = append(message, uint8(len(post.Meta.raw)))
	message = append(message, post.Meta.raw...)
	payloadLenBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(payloadLenBytes, uint16(len(*post.Payload)))
	message = append(message, payloadLenBytes...)
	message = append(message, *post.Payload...)
	fmt.Println("data:", message)
	err := peerFrom.Send(message)
	if err != nil {
		return err
	}
	return nil
}

func onGetNonce(payloadHash PayloadHash, peer *Peer) error {
	post := GetPost(payloadHash)
	if post == nil {
		return nil //um idk we don't have it. just ignore lol
	}
	message := []byte{uint8(PacketMultiNonce)}
	message = append(message, payloadHash[:]...)
	if len(post.Meta.raw) > 255 {
		panic("not long enough")
	}
	message = append(message, uint8(len(post.Meta.raw)))
	message = append(message, post.Meta.raw...)

	nonces := post.FlattenNonces()
	if len(nonces) > 65535 {
		panic("what on earth")
	}

	nonceCountBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(nonceCountBytes, uint16(len(nonces)))
	message = append(message, nonceCountBytes...)
	for i := 0; i < len(nonces); i++ {
		message = append(message, nonces[i][:]...)
	}

	fmt.Println("data:", message)
	err := peer.Send(message)
	if err != nil {
		return err
	}
	return nil
	//even if we dont have the payload, its ok to just send our best nonces for this payload hash
}
func onPacketMultiNonce(payloadHash PayloadHash, nonces []Nonce, meta Meta, peer *Peer) error {
	post := genPost(payloadHash, nonces, meta)

	if post.Acceptable() {
		message := []byte{uint8(PacketPayloadRequest)}
		message = append(message, payloadHash[:]...)
		fmt.Println("data:", message)
		err := peer.Send(message)
		if err != nil {
			return err
		}
	}
	return nil
	//huehuehue just call readPacketNonceUpdate in a loop
}
