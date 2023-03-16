package cmd

import (
	"fmt"
	"log"
	"os"
)

// handle errors
func handle(e error) {
	if e != nil {

		//log error if verbose
		debug(e)

		//buld message to show user
		msg := `something went wrong 😞`
		if !verbose {
			msg += `  try -v for more info`
		}

		//show message
		errorExit(msg)
	}
}

// wraps an error with context and handles it
func handleError(context string, err error) {
	if err != nil {
		handle(fmt.Errorf("%s: %w", context, err))
	}
}

// prints message to user and exits
func errorExit(a ...interface{}) {
	fmt.Println(a...)
	os.Exit(1)
}

// if -v, log detailed message
func debug(a ...interface{}) {
	if verbose {
		log.Println(a...)
	}
}

// if -v, log detailed message
func debugFmt(format string, a ...interface{}) {
	debug(fmt.Sprintf(format, a...))
}

// SliceContains returns true if a slice contains a string
func SliceContains(s *[]string, e string) bool {
	for _, str := range *s {
		if str == e {
			return true
		}
	}
	return false
}
