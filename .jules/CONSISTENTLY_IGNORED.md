# Consistently Ignored Changes

This file lists patterns of changes that have been consistently rejected by human reviewers. All agents MUST consult this file before proposing a new change. If a planned change matches any pattern described below, it MUST be abandoned.

---

## IGNORE: Premature Abstraction of Simple Logic

**- Pattern:** Extracting simple, single-use logic (like basic authentication checks) into middleware, separate handler functions, or complex abstractions.
**- Justification:** The project prefers inline logic for simplicity and readability when the logic is not reused across multiple endpoints. Premature abstraction adds unnecessary indirection.
**- Files Affected:** `chromo.go`, `task.go`

## IGNORE: Weak Typing for Validated Inputs

**- Pattern:** Validating inputs (e.g., URLs) inside a function while keeping the function signature accepting a primitive type (e.g., `string`).
**- Justification:** "Parse, don't validate." Functions handling sensitive operations (like command execution) should accept strongly-typed, pre-validated objects (e.g., `*url.URL`) rather than raw strings to enforce type safety and prevent injection vulnerabilities.
**- Files Affected:** `chromium.go`

## IGNORE: Fragmented Fixes for Broken Builds

**- Pattern:** Submitting multiple small, isolated pull requests to fix individual compilation errors or panics when the build is broken.
**- Justification:** A broken build blocks development. Submitting piecemeal fixes creates noise and delays resolution. Submit a single, comprehensive pull request that restores the build to a working state.
**- Files Affected:** All

## IGNORE: API Contract Changes in Refactors

**- Pattern:** Using helpers like `http.Error` that modify the response body (e.g., adding newlines) in refactors, potentially breaking clients or tests expecting exact string matches.
**- Justification:** Refactors must preserve external behavior. `http.Error` appends a newline, which changes the response body and may break strict clients or tests.
**- Files Affected:** `chromo.go`

## IGNORE: Incomplete Tooling Configuration

**- Pattern:** Adding configuration files (e.g., `.golangci.yml`) without updating the CI/CD pipeline (`.github/workflows/*.yml`) to execute the new tools.
**- Justification:** Tooling is only effective if enforced by CI. Adding config files without enforcement creates false confidence and "dead" configuration.
**- Files Affected:** `.golangci.yml`, `.github/workflows/*`

## IGNORE: Generated Artifacts in Pull Requests

**- Pattern:** Submitting generated files, installation scripts (e.g., `install_mise.sh`), or build artifacts.
**- Justification:** The repository should contain source code only. Installation scripts should be fetched from official sources or generated during the build process, not committed to the repo.
**- Files Affected:** `install_mise.sh`, `dist/*`, `bin/*`
