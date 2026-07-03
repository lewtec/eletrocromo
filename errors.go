package eletrocromo

import (
	"log"
)

// ReportError provides centralized error reporting for unexpected errors.
// It ensures that errors from operations like I/O writes or background tasks
// are not silently swallowed.
func ReportError(err error) {
	if err != nil {
		log.Printf("unexpected error: %v", err)
	}
}
