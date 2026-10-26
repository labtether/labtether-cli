package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/labtether/labtether-cli/internal/client"
)

// A failed remote command may already have emitted its structured result.
// Execute must not append a second JSON document and break stdout consumers.
var jsonResponseWritten bool

// Cobra may reject a command or flag before parsing --json. Resolve only the
// output choice here; Cobra still owns argument validation and command parsing.
func requestsJSONOutput(args []string) bool {
	command, _, _ := rootCmd.Find(args)
	requested := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		if !strings.HasPrefix(args[i], "--") {
			continue
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(args[i], "--"), "=")
		if name == "json" {
			if !hasValue {
				value = "true"
			}
			if parsed, err := strconv.ParseBool(value); err == nil {
				requested = parsed
			}
			continue
		}
		if flag := command.Flags().Lookup(name); flag != nil && flag.NoOptDefVal == "" && !hasValue {
			i++ // A string/numeric flag value may itself look like --json.
		}
	}
	return requested
}

func printMutationResult(response *client.V2Response, format string, args ...any) {
	printStatusResult(json.RawMessage(response.Data), format, args...)
}

func printStatusResult(data any, format string, args ...any) {
	if jsonOutput {
		printJSON(data)
		return
	}
	fmt.Fprintf(rootCmd.OutOrStdout(), format, args...)
}
