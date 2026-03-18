package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Browse and manage files on assets",
}

var filesLsCmd = &cobra.Command{
	Use:   "ls <asset> <path>",
	Short: "List files in a directory on an asset",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		resp, err := c.Get(fmt.Sprintf("/api/v2/assets/%s/files?path=%s", url.PathEscape(args[0]), url.QueryEscape(args[1])))
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		var entries []map[string]any
		json.Unmarshal(resp.Data, &entries)

		fmt.Printf("%-10s %-10s %-20s %s\n", "TYPE", "SIZE", "MODIFIED", "NAME")
		for _, e := range entries {
			fmt.Printf("%-10v %-10v %-20v %v\n",
				e["type"], e["size"], e["modified"], e["name"])
		}
		return nil
	},
}

var filesCatCmd = &cobra.Command{
	Use:   "cat <asset> <path>",
	Short: "Print file contents from an asset",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		resp, err := c.Get(fmt.Sprintf("/api/v2/assets/%s/files/read?path=%s", url.PathEscape(args[0]), url.QueryEscape(args[1])))
		if err != nil {
			return err
		}

		var data map[string]any
		json.Unmarshal(resp.Data, &data)

		if content, ok := data["content"].(string); ok {
			fmt.Print(content)
		}
		return nil
	},
}

func init() {
	filesCmd.AddCommand(filesLsCmd, filesCatCmd)
	rootCmd.AddCommand(filesCmd)
}
