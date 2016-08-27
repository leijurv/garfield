package main
import
(
"time"
"crypto/sha256"
"bytes"
"encoding/hex"
"fmt"
"math/rand"
)
type Post struct{
	payloadRaw []byte
	nonce [32]byte
	firstReceived time.Time
	mostRecentNonceUpdate time.Time
}
func (post *Post) hash() [32]byte{
	combined:=append(post.payloadRaw,post.nonce[:]...)
	return sha256.Sum256(combined)
}
func (post *Post) checkPossibleNonce(newNonce [32]byte) bool{
	if bytes.Compare(newNonce[:],post.nonce[:])==0{
		return false
	}
	newHash:=sha256.Sum256(append(post.payloadRaw,newNonce[:]...))
	oldHash:=post.hash()
	comparison:=bytes.Compare(newHash[:],oldHash[:])
	if comparison < 0{
		//new hash is less than old, so this nonce is an improvement
		return true
	}
	return false
}
func increment(nonce [32]byte) [32]byte{
	for i:=31; i>=0; i--{
		nonce[i]++
		if nonce[i]!=0{
			break
		}
	}
	return nonce
}
func randomNonce() [32]byte{
	rand.Seed(time.Now().UTC().UnixNano())
	fixedLengthIsBS:=make([]byte,32)
	rand.Read(fixedLengthIsBS)
	var nonce [32]byte
	copy(nonce[:],fixedLengthIsBS)
	return nonce
}
func (post *Post) mine(count int){
	currentHash:=post.hash()
	nonce:=randomNonce()
	for i:=0; i<count; i++{
		newHash:=sha256.Sum256(append(post.payloadRaw,nonce[:]...))
		if bytes.Compare(newHash[:],currentHash[:])<0{
			currentHash=newHash
			post.nonce=nonce
			fmt.Println("Nonce improvement, hash is now ",newHash)
		}
		nonce=increment(nonce)
	}
}
func (post *Post) onNewNonceReceived(newNonce [32]byte){
	if post.checkPossibleNonce(newNonce){
		fmt.Println("Updating post nonce from",hex.EncodeToString(post.nonce[:]),"to",hex.EncodeToString(newNonce[:]))
		post.nonce=newNonce
		//send out update here
	}
}
func main(){
	toMine:=Post{payloadRaw:[]byte("wewlad")}
	fmt.Println(toMine.hash())
	toMine.mine(10000000)
	fmt.Println(toMine.hash())
}