package utils

import (
	"fmt"
	"log"
)

// CheckForNilAndHandleError will check to see if an error is nil and handle
// the error if it exists.
func CheckForNilAndHandleError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

// NewErrorMessage creates a new error with a given message.
func NewErrorMessage(message string, err error) error {
	return fmt.Errorf("%s: %v", message, err)
}

// HandleError outputs error messages and exits the program.
func HandleError(message string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
		return
	}

	log.Fatal(message)
}
