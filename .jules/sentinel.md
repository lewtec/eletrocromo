## 2024-07-25 - Fix Command Injection in chromium.go

**Vulnerability:** The `LaunchChromium` function in `chromium.go` was vulnerable to command injection. The `url` parameter was being directly concatenated into the command string using `fmt.Sprintf`, which could allow an attacker to execute arbitrary commands by crafting a malicious URL.

**Learning:** This vulnerability existed because of improper input handling when constructing a command to be executed by the operating system. Directly concatenating user-provided input into a command string is a classic security anti-pattern.

**Prevention:** To prevent this type of vulnerability in the future, all user-provided input that is used in system commands must be sanitized and passed as separate arguments to the `exec.Command` function. Additionally, validating the URL scheme to only allow `http` and `https` adds another layer of defense.

## 2026-01-17 - Fix Timing Attack in Auth Token Comparison

**Vulnerability:** The authentication logic in `chromo.go` used variable-time string comparisons (`token == a.AuthToken`) to validate the authentication token. This vulnerability could allow an attacker to determine the valid token byte-by-byte by measuring the time it takes for the server to respond (timing attack).

**Learning:** Standard string comparison in Go (and many languages) fails fast; it stops as soon as a mismatch is found. This optimization leaks information about how much of the string matched. For sensitive data like authentication tokens, this side channel can be exploited.

**Prevention:** Always use constant-time comparison functions (like `crypto/subtle.ConstantTimeCompare`) when validating secrets, signatures, or tokens. This ensures the comparison time depends only on the length of the inputs, not their content.
