//go:build ignore

package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/hash_password.go <password>")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}
