## 2026-01-23 - Fix build and cleanup chromo.go

**Issue:** `chromo.go` failed to build due to a missing `net/url` import. It also contained a potential runtime panic if `App.Context` was nil, and included dead code (a dummy sleep task).
**Root Cause:** Likely an oversight during initial implementation or refactoring where `url` usage was added without the import, and debug code was left in.
**Solution:** Added the missing import, implemented a nil-check for `Context` defaulting to `Background`, and removed the useless background task.
**Pattern:** Always run the build (`go build`) before committing. Ensure public structs with optional fields (like `Context`) have safe defaults. Remove debug code.
