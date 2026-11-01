package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all hub objects",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/v2/search?q=%s", url.QueryEscape(args[0]))
		resp, err := c.Get(path)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}
		// The Hub leaves out each kind that has no match, so no match is {}.
		var data map[string]json.RawMessage
		if err := decodeResponseData(resp, &data); err != nil {
			return err
		}
		var results []map[string]any
		for _, field := range []string{"assets", "groups"} {
			raw, found := data[field]
			if !found {
				continue
			}
			rows, err := decodeRows(raw)
			if err != nil {
				return fmt.Errorf("decode response data: %s: %w", field, err)
			}
			results = append(results, rows...)
		}
		if len(results) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No results.")
			return nil
		}
		printTable(results, col("TYPE", "type"), col("ID", "id"), col("NAME", "name"),
			col("STATUS", "status"), col("PLATFORM", "platform"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
