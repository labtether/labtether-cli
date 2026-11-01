package cmd

import (
	"errors"
	"fmt"
	"sync"

	"github.com/spf13/cobra"
)

// usageError marks a bad argument that a command finds itself, so the CLI
// shows usage text as it does for Cobra's own command, flag and argument
// checks.
type usageError struct{ message string }

func (e usageError) Error() string { return e.message }

func usageErrorf(format string, args ...any) error {
	return usageError{message: fmt.Sprintf(format, args...)}
}

// commandStarted is set when a command's RunE begins. Errors before that point
// come from Cobra's parsing and validation; errors after it are Hub, network or
// remote failures, where usage text only hides the real message.
var (
	commandStarted         bool
	trackCommandsStartOnce sync.Once
)

func trackCommandStart(cmd *cobra.Command) {
	if run := cmd.RunE; run != nil {
		cmd.RunE = func(c *cobra.Command, args []string) error {
			commandStarted = true
			return run(c, args)
		}
	}
	for _, child := range cmd.Commands() {
		trackCommandStart(child)
	}
}

func showsUsage(err error) bool {
	var usage usageError
	return !commandStarted || errors.As(err, &usage)
}
