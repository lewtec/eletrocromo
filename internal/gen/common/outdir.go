package common

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// PrepareOutDir creates out or, with force, wipes an existing directory's contents.
func PrepareOutDir(out string, force bool) error {
	st, err := os.Stat(out)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return os.MkdirAll(out, 0o755)
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("%w: %s", ErrOutPathNotDir, out)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("%w: %s", ErrOutDirNotEmpty, out)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(out, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
