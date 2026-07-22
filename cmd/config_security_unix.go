//go:build !windows

package cmd

import "os"

func hardenConfigFile(path string) error {
	return os.Chmod(path, 0o600)
}

func replaceConfigFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
