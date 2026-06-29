package eletrocromo

import "log"

// ReportError provides a centralized mechanism for reporting unexpected errors
// across the application. This ensures consistent logging and provides a single
// integration point for future error tracking systems (like Sentry).
func ReportError(err error) {
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
}
