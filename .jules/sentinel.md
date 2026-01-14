## 2024-07-25 - Fix Command Injection in chromium.go

**Vulnerability:** The `LaunchChromium` function in `chromium.go` was vulnerable to command injection. The `url` parameter was being directly concatenated into the command string using `fmt.Sprintf`, which could allow an attacker to execute arbitrary commands by crafting a malicious URL.

**Learning:** This vulnerability existed because of improper input handling when constructing a command to be executed by the operating system. Directly concatenating user-provided input into a command string is a classic security anti-pattern.

**Prevention:** To prevent this type of vulnerability in the future, all user-provided input that is used in system commands must be sanitized and passed as separate arguments to the `exec.Command` function. Additionally, validating the URL scheme to only allow `http` and `https` adds another layer of defense.
