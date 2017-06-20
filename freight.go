package main

import (
	"fmt"
)

func saveNewPost(post *Post) {
	fmt.Println("SAVING NEW POST WITH PAYLOAD HASH", post.PayloadHash)
}
func saveNonceUpdate(post *Post) {
	fmt.Println("SAVING NONCE UPDATE FOR POST WITH PAYLOAD HASH", post.PayloadHash)
}
func initializeFreight() {

}
