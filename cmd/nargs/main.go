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
	includeNamedReturns := flag.Bool("named_returns", false, "Report unused named return arguments")
	includeReceivers := flag.Bool("receivers", true, "Report unused function receivers")

	flag.Parse()

	flags := nargs.Flags{
		IncludeTests:        *includeTests,
		SetExitStatus:       *setExitStatus,
		IncludeNamedReturns: *includeNamedReturns,
		IncludeReceivers:    *includeReceivers,
	}

	flag.Usage = usage

	results, exitWithCode, err := nargs.CheckForUnusedFunctionArgs(flag.Args(), flags)
	if err != nil {
		log.Printf("ERROR: failed to run %s, %v\n", os.Args[0], err)
	}

	for _, result := range results {
		log.Printf(result)
	}

	if exitWithCode {
		os.Exit(1)
	}
}
