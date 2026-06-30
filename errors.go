package eletrocromo

import "log"

// ReportError is the centralized error reporting function for the project.
// All unexpected errors (including those from write operations) must be passed
// to this function instead of being silently swallowed or logged directly at the call site.
func ReportError(err error) {
	if err != nil {
		log.Printf("unexpected error: %v", err)
	}
}
