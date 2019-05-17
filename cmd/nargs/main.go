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

	includeTests := flag.Bool("tests", true, "include test (*_test.go) files")
	setExitStatus := flag.Bool("set_exit_status", true, "Set exit status to 1 if any issues are found")

	flag.Parse()

	flags := nargs.Flags{
		IncludeTests:  *includeTests,
		SetExitStatus: *setExitStatus,
	}

	flag.Usage = usage

	if err := nargs.CheckForUnusedFunctionArgs(flag.Args(), flags); err != nil {
		log.Println(err)
	}
}
