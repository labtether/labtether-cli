package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "List and read files on a device",
}

var filesLsCmd = &cobra.Command{
	Use:   "ls <device> <path>",
	Short: "List a directory on a device",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		resp, err := c.Get(fmt.Sprintf("/api/v2/assets/%s/files?path=%s", pathSegment(args[0]), url.QueryEscape(args[1])))
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		rows, err := decodeTableRows(resp, "entries")
		if err != nil {
			return err
		}
		printTable(rows,
			colFormat("TYPE", "is_dir", func(value any) string {
				if value == true {
					return "directory"
				}
				return "file"
			}),
			col("SIZE", "size"),
			colFormat("MODIFIED", "mod_time", timeText),
			col("NAME", "name"))
		return nil
	},
}

var filesCatCmd = &cobra.Command{
	Use:   "cat <device> <path>",
	Short: "Print a file from a device",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		return c.Download(fmt.Sprintf("/api/v2/assets/%s/files/read?path=%s", pathSegment(args[0]), url.QueryEscape(args[1])), cmd.OutOrStdout())
	},
}

func init() {
	filesCmd.AddCommand(filesLsCmd, filesCatCmd)
	rootCmd.AddCommand(filesCmd)
}
