package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nexusriot/antiworld3/internal/crypto"
)

func main() {

	password := flag.String("password", "", "password string to encrypt")
	flag.Parse()

	if *password == "" {
		fmt.Println("missing required argument: password")
		os.Exit(2)
	}

	enc, err := crypto.Encrypt(*password)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error encrypting %s because of %s", *password, err.Error()))
	}
	fmt.Printf("Encrypted password: %s\n", enc)

	// verification
	dec, err := crypto.Decrypt(enc)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error verifying because of %s", err.Error()))
	}
	if *password != dec {
		fmt.Println("Verification failed! Decrypted password does not match!")
		os.Exit(2)
	} else {
		fmt.Println("Verification: OK")
	}
}
