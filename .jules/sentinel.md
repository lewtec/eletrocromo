## 2024-07-25 - Command Injection in chromium.go
**Vulnerability:** The `LaunchChromium` function in `chromium.go` was vulnerable to command injection. It used `fmt.Sprintf("--app=%s", url)` to construct a command-line argument. A URL containing spaces and malicious flags (e.g., `http://example.com --new-window --load-extension=/path/to/malicious/extension`) could be passed to `exec.Command`, allowing an attacker to inject arbitrary arguments into the browser process.

**Learning:** This vulnerability existed because user-controllable input (the URL) was directly concatenated into a command-line string instead of being passed as a distinct argument. The `exec.Command` function is designed to treat each argument in its string slice as a single, separate entity, which prevents this type of injection.

**Prevention:** To prevent similar issues, never build command strings using concatenation or string formatting with untrusted input. Always pass arguments to `exec.Command` as a slice of strings. Additionally, input validation should be implemented as a first line of defense; in this case, parsing the URL and checking its scheme adds another layer of security.
