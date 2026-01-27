## 2024-07-25 - Fix Command Injection in chromium.go

**Vulnerability:** The `LaunchChromium` function in `chromium.go` was vulnerable to command injection. The `url` parameter was being directly concatenated into the command string using `fmt.Sprintf`, which could allow an attacker to execute arbitrary commands by crafting a malicious URL.

**Learning:** This vulnerability existed because of improper input handling when constructing a command to be executed by the operating system. Directly concatenating user-provided input into a command string is a classic security anti-pattern.

**Prevention:** To prevent this type of vulnerability in the future, all user-provided input that is used in system commands must be sanitized and passed as separate arguments to the `exec.Command` function. Additionally, validating the URL scheme to only allow `http` and `https` adds another layer of defense.

## 2026-01-27 - Fix Broken Access Control in Auth Verification

**Vulnerability:** The `ServeHTTP` method in `chromo.go` allowed unauthorized access if the `App` structure was used without initializing `AuthToken`. An empty `AuthToken` (default state) matched an empty request token, granting access.

**Learning:** Authentication systems must "fail closed". Relying on upstream initialization (like `App.Run`) is risky if the component can be used independently. Uninitialized security controls should deny access, not allow it.

**Prevention:** Implement checks to ensure security configuration is valid before enforcing it. Treat uninitialized states as "access denied". Add unit tests for default/uninitialized states of security components.
