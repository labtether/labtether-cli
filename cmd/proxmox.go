package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var proxmoxCmd = &cobra.Command{
	Use:   "proxmox",
	Short: "Interact with Proxmox clusters",
}

var proxmoxClusterStatusCmd = &cobra.Command{
	Use:   "cluster-status",
	Short: "Show Proxmox cluster status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/proxmox/cluster/status")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var proxmoxResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List all Proxmox resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/proxmox/resources")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var proxmoxNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "List Proxmox nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/proxmox/nodes")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var proxmoxGetCmd = &cobra.Command{
	Use:   "get <vm>",
	Short: "Get VM/CT details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/proxmox/vms/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var proxmoxStartCmd = &cobra.Command{
	Use:   "start <vm>",
	Short: "Start a VM or container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/proxmox/vms/%s/start", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("VM %s start requested\n", args[0])
		return nil
	},
}

var proxmoxStopCmd = &cobra.Command{
	Use:   "stop <vm>",
	Short: "Stop a VM or container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/proxmox/vms/%s/stop", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("VM %s stop requested\n", args[0])
		return nil
	},
}

var proxmoxRestartCmd = &cobra.Command{
	Use:   "restart <vm>",
	Short: "Restart a VM or container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/proxmox/vms/%s/restart", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("VM %s restart requested\n", args[0])
		return nil
	},
}

var proxmoxCephStatusCmd = &cobra.Command{
	Use:   "ceph-status",
	Short: "Show Ceph storage status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/proxmox/ceph/status")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	proxmoxCmd.AddCommand(
		proxmoxClusterStatusCmd,
		proxmoxResourcesCmd,
		proxmoxNodesCmd,
		proxmoxGetCmd,
		proxmoxStartCmd,
		proxmoxStopCmd,
		proxmoxRestartCmd,
		proxmoxCephStatusCmd,
	)
	rootCmd.AddCommand(proxmoxCmd)
}
