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

		resp, err := c.Get(fmt.Sprintf("/api/v2/assets/%s/files?path=%s", pathSegment(args[0]), url.QueryEscape(args[1])))
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		var listing struct {
			Entries []struct {
				Name    string `json:"name"`
				Size    int64  `json:"size"`
				ModTime string `json:"mod_time"`
				IsDir   bool   `json:"is_dir"`
			} `json:"entries"`
		}
		if err := decodeResponseList(resp, "entries", &listing.Entries); err != nil {
			return err
		}

		fmt.Printf("%-10s %-10s %-20s %s\n", "TYPE", "SIZE", "MODIFIED", "NAME")
		for _, e := range listing.Entries {
			kind := "file"
			if e.IsDir {
				kind = "directory"
			}
			fmt.Printf("%-10s %-10d %-20s %s\n", kind, e.Size, e.ModTime, e.Name)
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

		return c.Download(fmt.Sprintf("/api/v2/assets/%s/files/read?path=%s", pathSegment(args[0]), url.QueryEscape(args[1])), cmd.OutOrStdout())
	},
}

func init() {
	filesCmd.AddCommand(filesLsCmd, filesCatCmd)
	rootCmd.AddCommand(filesCmd)
}
