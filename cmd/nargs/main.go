package main

import (
	"flag"
	"log"
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

	if err := nargs.CheckForUnusedFunctionArgs(flag.Args()); err != nil {
		log.Println(err)
	}
}
