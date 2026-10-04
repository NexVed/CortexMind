// Package repofiles confines source reads to the selected repository root.
package repofiles

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReadFile uses Go's traversal-resistant root API, including symlink/junction checks.
func ReadFile(root, name string) ([]byte, error) {
	directory, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	file, err := directory.Open(filepath.FromSlash(name))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	const maxBytes = 512 << 10
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return nil, fmt.Errorf("source file is not regular or exceeds 512 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if len(data) > maxBytes {
		return nil, fmt.Errorf("source file exceeds 512 KiB")
	}
	return data, err
}
