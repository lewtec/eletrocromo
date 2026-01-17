# Janitor's Journal

## 2026-01-17 - Fix Broken Build and Nil Context Panic
**Issue:** The project failed to compile due to a missing `net/url` import in `chromo.go`. Additionally, `App.Run` would panic if `App.Context` was not initialized by the caller.
**Root Cause:**
1.  Usage of `url.Parse` without importing `net/url`.
2.  `context.WithCancel` calls on `a.Context` without checking if it's nil.
**Solution:**
1.  Added `"net/url"` to imports.
2.  Added default initialization `if a.Context == nil { a.Context = context.Background() }`.
**Pattern:** Always ensure dependencies are imported and verify optional struct fields (like Context) are initialized before use to prevent runtime panics.
