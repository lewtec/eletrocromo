package eletrocromo

import "log"

// ReportError centralizes error reporting for the application.
// All unexpected errors must be passed to this function.
func ReportError(err error) {
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
}
