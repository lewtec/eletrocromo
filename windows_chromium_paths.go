package eletrocromo

// windowsExtraChromiumLikes returns absolute install paths for common
// Chromium-like browsers on Windows. localAppData is the LOCALAPPDATA
// directory (may be empty). Kept free of GOOS build tags so unit tests run
// on Linux CI; only the windows init hook prepends these to chromiumLikes.
//
// Paths use Windows separators intentionally: these strings are only ever
// passed to exec.LookPath on Windows.
func windowsExtraChromiumLikes(localAppData string) []string {
	extra := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		`C:\Program Files (x86)\BraveSoftware\Brave-Browser\Application\brave.exe`,
	}
	if localAppData == "" {
		return extra
	}
	// Per-user installs (Chrome's default on modern Windows) live under
	// LocalAppData, not Program Files, and are usually absent from PATH.
	return append(extra,
		localAppData+`\Google\Chrome\Application\chrome.exe`,
		localAppData+`\Microsoft\Edge\Application\msedge.exe`,
		localAppData+`\BraveSoftware\Brave-Browser\Application\brave.exe`,
	)
}
