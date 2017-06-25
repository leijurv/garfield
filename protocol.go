package main

import (
	//"crypto/sha256"
	"encoding/binary"
	//"time"
	"errors"
)

const (
	FlagHasPayload = 1 << iota
	FlagHasNonces
	FlagAcceptableScore
	FlagAcceptableMeta
)

var dedup map[Nonce]bool

func onNonceUpdateReceived(postPayloadHash PayloadHash, meta Meta, newNonce Nonce, tiene byte, peer *Peer) error {
	//TODO maybe send payloadBodyHash + meta instead of postPayloadHash
	_, newWork := postPayloadHash.Sentiment(newNonce)
	_, ok := calcDepth(newWork)
	if !ok {
		//we just received a nonce update resulting in work less than minimum depth
		return PeerRemovalError{errors.New("stop lying to me")}
	}

	_, already := dedup[newNonce] //once we know that the pow is acceptable, let's check if we've already seen this nonce
	dedup[newNonce] = true        //only do this after the pow is ok so that they cant spam our ram
	if already {
		return nil
	}

	message := []byte{uint8(PacketNonceUpdate)}
	message = append(message, postPayloadHash[:]...)
	message = append(message, newNonce[:]...)

	if len(meta.raw) > 255 {
		panic("not long enough")
	}
	message = append(message, meta.Write()...)

	if !meta.Verify() { //we don't care
		if tiene&FlagHasNonces == 0 {
			//we just received from someone who also doesn't know what they are doing
			//if we rebroadcast, it will go back to them with the same flags, and we will have an infinite loop
			if tiene&FlagAcceptableMeta != 0 {
				//they do like the meta though
				//hopefully they will have the nonces soon because they like the meta
				//let's not put any effort into trying to get these nonces because, we, don't, care
				return nil
			} else {
				return nil //they don't have the nonces, and in fact don't care, and we also don't care. lol
				//hmmmmm, bad peer! you're not supposed to rebroadcast if you don't have nonces or meta... UNLESS you received from someone with nonces AND meta
			}
			return nil
		}
		var flags uint8
		post, err := postManager.PostBacking.GetPost(postPayloadHash)
		if err != nil && err != ErrPostMissing {
			return err
		}
		if post != nil && post.HasPayload() { //why check payload? because maybe the meta used to be acceptable but now isn't, but we still have the payload from earlier I guess
			//don't set HasNonces because we aren't keeping track of the nonces so it would be deceiving
			flags |= FlagHasPayload
		}
		message = append(message, flags)
		Broadcast(message)
		return nil
	}
	post, err := postManager.PostBacking.GetPost(postPayloadHash)
	if err != nil && err != ErrPostMissing {
		return err
	}
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
			return nil
		}
	} else {
		//TODO honestly maybe just broadcast a getnonce to EVERY peer...

		//TODO rebroadcast this improvement with proper flags here
		if tiene&FlagHasNonces != 0 {
			//ask the peer we received this from for the nonces for the post because we don't have it
			message := []byte{uint8(PacketGetNonce)}
			message = append(message, postPayloadHash[:]...)
			Debug.Println("data:", message)
			err := peer.Send(message)
			if err != nil {
				return err //this error gets passed all the way back up through the packet handler to peer.Listen, so if writing to this peer fails, we will actually disconnect and remove them
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
				//solution: don't do anything
				//if we have at least one other peer who cares, we'll get the same update from them
				//TODO broadcast a getnonce to everyone lol
			}
		}
	}
	return nil
}

func onPayloadReceived(payloadHash PayloadHash, meta Meta, payloadBodyHash [32]byte, payloadBody Payload) error {
	Debug.Println("Post contents:", payloadBody)
	post, err := postManager.PostBacking.GetPost(payloadHash)
	if err != nil && err != ErrPostMissing {
		return err
	}
	if post != nil {
		Debug.Println("Cool, a payload")
		post.payloadReceived(payloadBody)
	} else {
		Warning.Println("While appreciated, I did not ask for contents of", payloadHash, "so I don't have its nonces so I can't accept it")
	}
	return nil
}

func onPayloadRequested(payloadHash PayloadHash, peerFrom *Peer) error {
	//TODO pow
	post, err := postManager.PostBacking.GetPost(payloadHash)
	if err != nil && err != ErrPostMissing {
		return err
	}
	if post == nil {
		return nil //um idk we don't have it. just ignore lol
	}
	if !post.HasPayload() {
		return nil
	}

	Debug.Println("Sending contents of ", payloadHash)
	//man I wish go was better at appending mulitple arrays. lol im probbaly doing something wrong here. BUT HEY, IT WORKS

	message := []byte{uint8(PacketPayload)}
	message = append(message, payloadHash[:]...)
	message = append(message, post.Meta.Write()...)
	payloadLenBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(payloadLenBytes, uint16(len(*post.Payload)))
	message = append(message, payloadLenBytes...)
	message = append(message, *post.Payload...)
	Debug.Println("data:", message)
	err = peerFrom.Send(message)
	if err != nil {
		return err
	}
	return nil
}

func onGetNonce(payloadHash PayloadHash, peer *Peer) error {
	post, err := postManager.PostBacking.GetPost(payloadHash)
	if err != nil && err != ErrPostMissing {
		return err
	}
	if post == nil {
		return nil //um idk we don't have it. just ignore lol
	}
	message := []byte{uint8(PacketMultiNonce)}
	message = append(message, payloadHash[:]...)

	message = append(message, post.Meta.Write()...)

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

	Debug.Println("data:", message)
	err = peer.Send(message)
	if err != nil {
		return err
	}
	return nil
	//even if we dont have the payload, its ok to just send our best nonces for this payload hash
}

func onPacketMultiNonce(payloadHash PayloadHash, nonces []Nonce, meta Meta, peer *Peer) error {
	post := genPost(payloadHash, nonces, meta)
	if post.Acceptable() { //genPost inserts all these awesome nonces and works where they should go. now we can check if the score is acceptable
		message := []byte{uint8(PacketPayloadRequest)}
		message = append(message, payloadHash[:]...)
		Debug.Println("data:", message)
		err := peer.Send(message)
		if err != nil {
			return err
		}
		return nil
	} else {
		return nil
	}
}
