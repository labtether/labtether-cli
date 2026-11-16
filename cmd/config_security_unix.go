//go:build !windows

package cmd

import "os"

func hardenConfigFile(path string) error {
	return os.Chmod(path, 0o600)
}

func replaceConfigFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// createConfigTemp makes the config's temporary copy, mode 0600 from the
// start.
func createConfigTemp(dir string) (*os.File, error) {
	return os.CreateTemp(dir, ".config-*.tmp")
}
