package main

import (
	"bytes"
	"crypto/sha256"
	//"fmt"
	"sync"
	//"time"
)

type PayloadHash [32]byte

type Nonce [32]byte

type Nonces [][]Nonce

type Work [][][32]byte

type Payload []byte

type Post struct {
	PayloadHash PayloadHash
	Nonces      Nonces
	Meta        Meta
	Payload     *Payload
	work        Work
	lock        sync.Mutex
}

type Meta struct {
	raw  []byte
	Data map[string]interface{}
}

var requestedPosts map[PayloadHash]*Post
var reqLock sync.Mutex

func getReq(payloadHash PayloadHash) *Post {
	reqLock.Lock()
	defer reqLock.Unlock()
	post, ok := requestedPosts[payloadHash]
	if !ok {
		return nil
	}
	return post
}
func GetPost(payloadHash PayloadHash) *Post {
	post := getReq(payloadHash)
	if post != nil {
		return post
	}
	//TODO
	return nil
}
func (meta Meta) Verify() bool {
	//TODO
	return true
}
func (hash PayloadHash) Sentiment(nonce Nonce) (bool, [32]byte) {
	positive := hash.Work(nonce, true)
	negative := hash.Work(nonce, false)
	if bytes.Compare(positive[:], negative[:]) < 0 { //return true if positive is less than negative
		return true, positive
	} else {
		return false, negative
	}
}

func (hash PayloadHash) Work(nonce Nonce, sentiment bool) [32]byte { // hash(hash(hash(payolad)+"up")+nonce)
	var sent []byte
	if sentiment {
		sent = []byte("up")
	} else {
		sent = []byte("down")
	}
	tmp := sha256.Sum256(append(hash[:], sent...))
	res := sha256.Sum256(append(tmp[:], nonce[:]...))
	return res
}
func (payload Payload) BodyHash() [32]byte {
	return sha256.Sum256(payload)
}
func (post *Post) PayloadBodyHash() [32]byte {
	if !post.HasPayload() {
		panic("HEY")
	}
	return post.Payload.BodyHash()
}
func (post *Post) FlattenNonces() []Nonce {
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
	reqLock.Lock()
	defer reqLock.Unlock()

	post, ok := requestedPosts[payloadHash]
	if !ok {
		post := &Post{
			PayloadHash: payloadHash,
			Meta:        meta,
			Payload:     nil,
		}
		requestedPosts[payloadHash] = post
	}
	for i := 0; i < len(nonces); i++ {
		post.insertIfImprovement(nonces[i])
	}
	return post
}
func (post *Post) insertIfImprovement(nonce Nonce) bool {
	return true
}

// CheckPossibleNonce compares the new nonce hash with the old one
/*func (post *Post) CheckPossibleNonce(newNonce [32]byte) int {
	if bytes.Equal(newNonce[:], post.nonce[:]) {
		return 0
	}
	newHash := post.hashNonce(newNonce)
	oldHash := post.Hash()
	comparison := bytes.Compare(newHash[:], oldHash[:])
	return comparison
}*/

// Insert adds a post to the list
/*func (post *Post) Insert() {
	postsLock.Lock()
	posts[post.PayloadHash()] = post
	postsLock.Unlock()
	saveNewPost(post)
}*/

// Mine does something
/*func (post *Post) Mine(count int) {
	currentHash := post.Hash()
	nonce := randomNonce()
	for i := 0; i < count; i++ {
		newHash := sha256.Sum256(append(post.PayloadRaw, nonce[:]...))
		if bytes.Compare(newHash[:], currentHash[:]) < 0 {
			currentHash = newHash
			postsLock.Lock()
			post.updateNonce(nonce)
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
}*/
