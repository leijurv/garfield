package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sync"
	//"time"
	"encoding/hex"
)

const bucketSize = 16

type Nonce [32]byte
type Nonces [][]Nonce
type Work [][][32]byte

type Post struct {
	PayloadHash PayloadHash
	Nonces      Nonces
	Meta        Meta
	Payload     *Payload
	work        Work
	lock        sync.Mutex
}

var (
	postManager = &PostManager{
		PostBacking: &MemoryPostCache{},
		PayloadBacking: &MemoryPayloadCache{},
	}
)

func (post *Post) PayloadBodyHash() [32]byte {
	return post.Payload.BodyHash()
}
func (post *Post) FlattenNonces() []Nonce {
	post.lock.Lock()
	defer post.lock.Unlock()
	count := 0
	for i := 0; i < len(post.Nonces); i++ {
		count += len(post.Nonces[i])
	}
	res := make([]Nonce, count)
	count = 0
	for i := 0; i < len(post.Nonces); i++ {
		for j := 0; j < len(post.Nonces[i]); j++ {
			res[count] = post.Nonces[i][j]
			count++
		}
	}
	return res
}
func (post *Post) HasPayload() bool {
	return post.Payload != nil
}
func (post *Post) payloadReceived(payload Payload) {
	post.lock.Lock()
	defer post.lock.Unlock()
	if post.HasPayload() {
		return
	}
	if !post.Acceptable() {
		return
	}
	payloadBodyHash := payload.BodyHash()
	chk := sha256.Sum256(append(payloadBodyHash[:], post.Meta.raw...))
	if !bytes.Equal(chk[:], post.PayloadHash[:]) {
		return
	}
	post.Payload = &payload
}

/*func doWork(payloadHash PayloadHash, nonces Nonces) Work {
	res := make(Work, len(nonces))
	for i := 0; i < len(nonces); i++ {
		res[i] = make([][32]byte, len(nonces[i]))
		for j := 0; j < len(nonces[i]); j++ {
			_, work := payloadHash.Sentiment(nonces[i][j])
			res[i][j] = work
		}
	}
	return res
}*/
func (post *Post) AcceptableScore() bool {
	score := post.Score()
	return score > 1000
}
func (post *Post) Acceptable() bool {
	return post.Meta.Verify() && post.AcceptableScore()
}
func (post *Post) Score() int {
	//TODO
	return 0
}
func genPost(payloadHash PayloadHash, nonces []Nonce, meta Meta) *Post {
	if !meta.Verify() {
		panic("no")
	}
	post, _ := postManager.PostBacking.GetPost(payloadHash)
	if post == nil {
		post := &Post{
			PayloadHash: payloadHash,
			Meta:        meta,
			Payload:     nil,
		}
		postManager.PostBacking.WritePost(payloadHash, post) //do all this in the lock
	}

	for i := 0; i < len(nonces); i++ {
		post.insertIfImprovement(nonces[i]) //this may be inefficient, but to do otherwise would be huge code duplication
		//go post.insertIfImprovement LOLLOLOLLOLLOL
	}
	return post
}
func (post *Post) insertIfImprovement(nonce Nonce) bool { //this func only returns whether or not it actually improved, doesn't do anything else like rebroadcasting or saving
	_, newWork := post.PayloadHash.Sentiment(nonce)
	depth, ok := calcDepth(newWork)
	if !ok {
		return false //TODO maybe return an error this is dumb it doesn't even satisfy minimum depth so its malicious maybe
	}
	//^ try and do as much processing as possible outside of the lock.
	post.lock.Lock()
	defer func() { //why put both of these in an anonymous defer? to guarantee that post.verify() is called and returns BEFORE we unlock the post since something else could immediately start messing with it
		defer post.lock.Unlock()
		post.verify() //rerun afterwards to make sure we didn't mess up anything
	}()
	post.verify()                   //call this WITHIN the lock but AFTER we defer the unlock
	for len(post.Nonces) <= depth { //we don't already have any at this depth, this'll be the first =D
		post.Nonces = append(post.Nonces, make([]Nonce, 0))
		post.work = append(post.work, make([][32]byte, 0))
	}
	if len(post.work[depth]) >= bucketSize {
		if len(post.work[depth]) > bucketSize {
			panic("i should maybe stop putting these redundant checks everywhere")
		}
		worstWork := post.work[depth][0]
		index := 0
		for i := 1; i < bucketSize; i++ {
			if bytes.Compare(worstWork[:], post.work[depth][i][:]) < 0 {
				worstWork = post.work[depth][i]
				index = i
			}
		}
		//get the WORST one we currently have. replacing that one with the better option will result in the most improvement, and will maintain the invariant of the 16 best at this depth
		if bytes.Compare(newWork[:], post.work[depth][index][:]) < 0 { //this is a < not a <= so as not to replace with the same thing.
			//ding ding we have a winner
			Debug.Println("Replacing", post.work[depth][index][:], "with lower", newWork[:])
			chk1, _ := calcDepth(post.work[depth][index])
			chk2, _ := calcDepth(newWork)
			if chk1 != chk2 || chk1 != depth || chk2 != depth {
				panic("how on earth did this happen")
			}
			post.work[depth][index] = newWork
			post.Nonces[depth][index] = nonce
			return true
		}
		Debug.Println(newWork[:], "did not improve on ANY of", post.work[depth])
		//if we can't improve on the worst at this depth, then fail
		return false
	} else {
		Debug.Println("Adding new work because I have space for it", newWork[:])
		post.work[depth] = append(post.work[depth], newWork)
		post.Nonces[depth] = append(post.Nonces[depth], nonce)
		return true
	}

}
func (post *Post) verify() {
	verifySanity(post.Nonces, post.work)
}
func verifySanity(nonces Nonces, work Work) { //only call with lock, obviously
	if len(nonces) != len(work) {
		panic("INSANE")
	}
	for i := 0; i < len(nonces); i++ {
		if len(nonces[i]) != len(work[i]) {
			panic("INSANE")
		}
		if len(nonces[i]) > bucketSize {
			panic("INSANE")
		}
		if len(nonces[i]) == 0 && i == len(nonces)-1 { //can't END with an empty segment; empty segments in the middle are ok (but unlikely)
			panic("INSANE")
		}
		for j := 0; j < len(nonces[i]); j++ {
			depth, ok := calcDepth(work[i][j])
			if !ok {
				panic("INSANE")
			}
			if depth != i {
				panic("INSANE")
			}
		}
	}
}
func calcDepth(work [32]byte) (int, bool) { //forgive me
	const minWorkHex = 5
	str := hex.EncodeToString(work[:])
	if len(str) != 64 {
		panic("wtf")
	}
	pos := 0
	for str[pos] == '0' {
		pos++
	}
	depth := pos - minWorkHex - 1
	if depth < 0 {
		return -1, false
	}
	return depth, true
}

// Mine does something
func (post *Post) Mine(count int) {
	nonce := randomNonce()
	for i := 0; i < count; i++ {
		if post.insertIfImprovement(nonce) { //TODO this is mining for both sentiments lol
			_, newWork := post.PayloadHash.Sentiment(nonce)
			Debug.Println("Nonce improvement, hash is now ", newWork[:])
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
