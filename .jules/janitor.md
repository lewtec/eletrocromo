# Janitor Journal

## 2026-01-31 - Remove duplicate chromium entry

**Issue:** The `chromiumLikes` slice in `chromium.go` contained the string "chromium" twice.
**Root Cause:** Redundant data entry, likely a copy-paste error.
**Solution:** Removed the second occurrence of "chromium" from the list.
**Pattern:** Duplicate data in configuration/lists.

## 2026-02-03 - Refactor ServeHTTP Error Handling

**Issue:** `ServeHTTP` used verbose `w.WriteHeader` + `fmt.Fprintf` and ignored return values (`_, _ =`).
**Root Cause:** Manual implementation of standard error responses.
**Solution:** Replaced with `http.Error` for cleaner code and correct header handling.
**Pattern:** Use standard library helpers (e.g., `http.Error`) to simplify common tasks.
