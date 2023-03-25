package main

import (
	"fmt"
	"log"
	"os"

	"github.com/FlorinBalint/exif_reader/metadata"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("please give filename as argument")
	}
	fname := os.Args[1]

	md, err := metadata.FromPhoto(fname)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nMetadata:\n%v", md)
}
