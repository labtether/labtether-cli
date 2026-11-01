package cmd

import (
	"fmt"
	"strings"
)

// remoteExecResult is one command result from the Hub. Current Hubs send
// status (succeeded or failed) and exit_code only when the agent reported it.
// Hubs before that send exit_code without status. A command that never ran
// has error and message instead.
type remoteExecResult struct {
	Status   string `json:"status,omitempty"`
	ExitCode *int   `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
}

type multiExecResponse struct {
	Results map[string]remoteExecResult `json:"results"`
}

// known reports whether the result says how the command ended at all.
func (r remoteExecResult) known() bool {
	return r.Error != "" || strings.TrimSpace(r.Status) != "" || r.ExitCode != nil
}

func (r remoteExecResult) failed() bool {
	if r.Error != "" {
		return true
	}
	if status := strings.TrimSpace(r.Status); status != "" {
		return !strings.EqualFold(status, "succeeded")
	}
	return r.ExitCode == nil || *r.ExitCode != 0
}

// reason is the Hub's explanation for a command that never ran.
func (r remoteExecResult) reason() string {
	if message := strings.TrimSpace(r.Message); message != "" {
		return message
	}
	return strings.TrimSpace(r.Error)
}

// failure builds the CLI error for a failed result.
func (r remoteExecResult) failure() error {
	switch {
	case r.Error != "":
		return remoteCommandError{message: "remote command failed: " + r.reason(), exitCode: 1}
	case r.ExitCode != nil:
		return remoteCommandError{
			message:  fmt.Sprintf("remote command failed with exit code %d", *r.ExitCode),
			exitCode: processExitCode(*r.ExitCode),
		}
	default:
		return remoteCommandError{message: "remote command failed; the Hub did not report an exit code", exitCode: 1}
	}
}

// remoteCommandError is a remote command that did not succeed. The CLI exits
// with exitCode instead of mapping the message to an exit code.
type remoteCommandError struct {
	message  string
	exitCode int
}

func (e remoteCommandError) Error() string { return e.message }

// processExitCode passes a remote exit code through when a process exit
// status can carry it (1 to 255). Anything else, such as a large Windows
// code, a negative signal code or 0 on a failed result, becomes 1.
func processExitCode(code int) int {
	if code >= 1 && code <= 255 {
		return code
	}
	return 1
}
