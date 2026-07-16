//go:build windows

package eletrocromo

// Windows browsers are often installed under Program Files but not on PATH.
// The base list only has the x86 Edge path; prepend the common 64-bit install
// locations so GetChromium finds Chrome, Edge, and Brave without PATH entries.
func init() {
	chromiumLikes = append([]string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		`C:\Program Files (x86)\BraveSoftware\Brave-Browser\Application\brave.exe`,
	}, chromiumLikes...)
}
