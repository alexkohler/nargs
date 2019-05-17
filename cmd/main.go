package main

import (
	"flag"
	"log"
	"math/rand"
	"os"

	"github.com/alexkohler/nargs"
)

func usage() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("\nnargs [flags] # runs on package in current directory\n")
	log.Printf("\nnargs [flags] [packages]\n")
	log.Printf("Flags:\n")
	flag.PrintDefaults()
}
func main() {

	// Remove log timestamp
	log.SetFlags(0)

	maxLength := flag.Uint("l", 5, "maximum number of lines for a naked return function")
	flag.Usage = usage
	flag.Parse()
	i := rand.Int()

	if err := nargs.CheckForUnusedFunctionArgs(flag.Args(), maxLength, i); err != nil {
		log.Println(err)
	}
}
