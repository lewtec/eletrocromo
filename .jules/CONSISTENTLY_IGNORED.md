# Consistently Ignored Changes

This file lists patterns of changes that have been consistently rejected by human reviewers. All agents MUST consult this file before proposing a new change. If a planned change matches any pattern described below, it MUST be abandoned.

---

## IGNORE: Simplistic Fixes for Command Injection in `chromium.go`

**- Pattern:** Do not submit pull requests that attempt to fix the command injection vulnerability in `chromium.go` with simplistic solutions.

**- Justification:** This vulnerability is more complex than it appears. The `LaunchChromium` function is inherently dangerous because it constructs a command line argument with user-supplied input. Simply escaping characters is not sufficient, as different operating systems and shells have different parsing rules. Rejected pull requests have failed to account for this complexity.

**- Required Approach:** An acceptable fix MUST involve a multi-layered approach:
    1.  **Strict URL Validation:** The input `url` string must be rigorously validated to ensure it conforms to the `http` or `https` schemes.
    2.  **Argument Separation:** The `--app` flag and the URL must be passed as separate arguments to `exec.Command`. Do not concatenate them into a single string.
    3.  **Consider Edge Cases:** The solution must account for URLs that contain special characters, query parameters, and fragments.

**- Files Affected:** `chromium.go`
