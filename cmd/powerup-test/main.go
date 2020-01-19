package main

import (
	"fmt"
	"log"
	"os"

	powerup "github.com/dgparker/vegeta-powerup"
)

func main() {
	targets, err := powerup.Absorb(os.Getenv("COLLECTION_PATH"), os.Getenv("ENV_PATH"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(targets)
}
