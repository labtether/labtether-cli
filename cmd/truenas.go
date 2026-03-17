package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var truenasCmd = &cobra.Command{
	Use:   "truenas",
	Short: "Interact with TrueNAS systems",
}

var truenasGetCmd = &cobra.Command{
	Use:   "get <sys>",
	Short: "Get TrueNAS system details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var truenasPoolsCmd = &cobra.Command{
	Use:   "pools <sys>",
	Short: "List storage pools on a TrueNAS system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/" + args[0] + "/pools")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var truenasDatasetsCmd = &cobra.Command{
	Use:   "datasets <sys>",
	Short: "List datasets on a TrueNAS system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/" + args[0] + "/datasets")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var truenasSharesCmd = &cobra.Command{
	Use:   "shares <sys>",
	Short: "List shares on a TrueNAS system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/" + args[0] + "/shares")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	truenasCmd.AddCommand(truenasGetCmd, truenasPoolsCmd, truenasDatasetsCmd, truenasSharesCmd)
	rootCmd.AddCommand(truenasCmd)
}
