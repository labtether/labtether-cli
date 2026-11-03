package cmd

import (
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"
)

// map was called topology before 2.0, with blast-radius for downstream and
// edges for connections. The old names stay as hidden aliases.
var mapCmd = &cobra.Command{
	Use:     "map",
	Aliases: []string{"topology"},
	Short:   "Show how devices connect",
	Example: "  labtether-cli map connections pve01\n" +
		"  labtether-cli map downstream pve01\n" +
		"  labtether-cli map upstream nas01",
}

// mapQueryCmd builds a map subcommand that prints the Hub's JSON for one device.
func mapQueryCmd(use string, aliases []string, short, route, param string) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			resp, err := c.Get(route + "?" + param + "=" + url.QueryEscape(args[0]))
			if err != nil {
				return err
			}
			printJSON(json.RawMessage(resp.Data))
			return nil
		},
	}
}

func init() {
	mapCmd.AddCommand(
		mapQueryCmd("connections <device>", []string{"edges"}, "List a device's connections",
			"/api/v2/edges", "asset_id"),
		mapQueryCmd("dependencies <device>", nil, "List a device's service dependencies",
			"/api/v2/dependencies", "asset_id"),
		mapQueryCmd("downstream <device>", []string{"blast-radius"}, "Show the connections downstream of a device",
			"/api/v2/edges/tree", "root"),
		mapQueryCmd("upstream <device>", nil, "Show the connections upstream of a device",
			"/api/v2/edges/ancestors", "id"),
	)
	rootCmd.AddCommand(mapCmd)
}
