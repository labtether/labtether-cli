package cmd

import (
	"errors"
	"strings"
)

// Exit codes, as listed in the CLI guide.
const (
	exitFailed   = 1 // the command failed
	exitAuth     = 2 // the hub refused the API key (401 or 403)
	exitNotFound = 3 // the device is offline or the object is missing (409 or 404)
	exitConfig   = 4 // the CLI's own settings are missing or invalid
)

// configError is a problem with the CLI's own settings, found before any
// request: the hub URL, the API key or key file, the CA file or the config
// file. It exits 4.
type configError struct{ err error }

func (e configError) Error() string { return e.err.Error() }
func (e configError) Unwrap() error { return e.err }

func asConfigError(err error) error {
	if err == nil {
		return nil
	}
	var already configError
	if errors.As(err, &already) {
		return err
	}
	return configError{err: err}
}

// notFoundError is an object the CLI looks up itself and does not find, such
// as an unknown connector ID. It exits 3, like the hub's own not-found answers.
type notFoundError struct{ message string }

func (e notFoundError) Error() string { return e.message }

func exitCodeFor(err error) int {
	// A failed remote command exits with its own code when known.
	var remote remoteCommandError
	if errors.As(err, &remote) {
		return remote.exitCode
	}
	var config configError
	if errors.As(err, &config) {
		return exitConfig
	}
	var missing notFoundError
	if errors.As(err, &missing) {
		return exitNotFound
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "(status 401)") || strings.Contains(text, "(status 403)"):
		return exitAuth
	case strings.Contains(text, "(status 409)") || strings.Contains(text, "(status 404)"):
		return exitNotFound
	default:
		return exitFailed
	}
}
