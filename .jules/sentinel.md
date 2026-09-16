## 2024-07-25 - Fix Command Injection in chromium.go

**Vulnerability:** The `LaunchChromium` function in `chromium.go` was vulnerable to command injection. The `url` parameter was being directly concatenated into the command string using `fmt.Sprintf`, which could allow an attacker to execute arbitrary commands by crafting a malicious URL.

**Learning:** This vulnerability existed because of improper input handling when constructing a command to be executed by the operating system. Directly concatenating user-provided input into a command string is a classic security anti-pattern.

**Prevention:** To prevent this type of vulnerability in the future, all user-provided input that is used in system commands must be sanitized and passed as separate arguments to the `exec.Command` function. Additionally, validating the URL scheme to only allow `http` and `https` adds another layer of defense.

## 2026-01-23 - Mitigate Timing Attack in Auth Token Verification

**Vulnerability:** The `ServeHTTP` method in `chromo.go` used a standard string comparison (`==`) to verify the authentication token. This operation is not constant-time and terminates early upon a mismatch, allowing an attacker to deduce the token byte-by-byte by measuring the response time (timing attack). Additionally, `App.Run` would panic if `App.Context` was nil, causing a Denial of Service.

**Learning:** String comparison operators in Go (and many languages) are optimized for speed, not security. They return false as soon as a difference is found. For cryptographic secrets, this optimization leaks information. Robustness is also a security feature; uninitialized structs should not cause crashes.

**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing secrets, hashes, or tokens. Ensure that public APIs gracefully handle default/nil values (e.g., initializing a nil context to `context.Background()`).
