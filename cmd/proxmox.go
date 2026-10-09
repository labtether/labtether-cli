package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labtether/labtether-cli/internal/client"

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
		resp, err := c.Get("/api/v2/proxmox/cluster/resources")
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
		resp, err := c.Get("/api/v2/proxmox/cluster/resources")
		if err != nil {
			return err
		}
		var resources struct {
			Resources []map[string]any `json:"resources"`
		}
		if err := json.Unmarshal(resp.Data, &resources); err != nil {
			return fmt.Errorf("decode Proxmox resources: %w", err)
		}
		nodes := make([]map[string]any, 0)
		for _, resource := range resources.Resources {
			if resource["type"] == "node" {
				nodes = append(nodes, resource)
			}
		}
		printJSON(nodes)
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
		resp, err := c.Get("/api/v2/proxmox/assets/" + pathSegment(args[0]) + "/details")
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
		if err := requestProxmoxPowerAction(c, args[0], "start"); err != nil {
			return err
		}
		fmt.Printf("Proxmox asset %s started\n", args[0])
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
		if err := requestProxmoxPowerAction(c, args[0], "stop"); err != nil {
			return err
		}
		fmt.Printf("Proxmox asset %s stopped\n", args[0])
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
		if err := requestProxmoxPowerAction(c, args[0], "reboot"); err != nil {
			return err
		}
		fmt.Printf("Proxmox asset %s rebooted\n", args[0])
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

func requestProxmoxPowerAction(c *client.Client, assetID, verb string) error {
	c.HTTPClient.Timeout = hubLongActionTimeout
	resp, err := c.Get("/api/v2/assets/" + pathSegment(assetID))
	if err != nil {
		return err
	}
	var details struct {
		Asset struct {
			Source string `json:"source"`
			Type   string `json:"type"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(resp.Data, &details); err != nil {
		return fmt.Errorf("decode asset details: %w", err)
	}
	if !strings.EqualFold(details.Asset.Source, "proxmox") {
		return fmt.Errorf("asset %q is not a Proxmox asset", assetID)
	}
	prefix := ""
	switch strings.ToLower(details.Asset.Type) {
	case "vm":
		prefix = "vm"
	case "container":
		prefix = "ct"
	default:
		return fmt.Errorf("asset %q is not a Proxmox VM or container", assetID)
	}
	actionID := prefix + "." + verb
	result, err := c.Post("/api/v2/connectors/proxmox/actions/"+actionID+"/execute", map[string]string{"target_id": assetID})
	if err != nil {
		return err
	}
	var action struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(result.Data, &action); err != nil {
		return fmt.Errorf("decode Proxmox action result: %w", err)
	}
	if !strings.EqualFold(action.Status, "succeeded") {
		if action.Message != "" {
			return fmt.Errorf("Proxmox action failed: %s", action.Message)
		}
		return fmt.Errorf("Proxmox action failed with status %q", action.Status)
	}
	return nil
}
