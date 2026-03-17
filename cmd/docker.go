package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Manage Docker containers across assets",
}

var dockerHostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "List Docker hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		resp, err := c.Get("/api/v2/docker/hosts")
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		var hosts []map[string]any
		json.Unmarshal(resp.Data, &hosts)

		fmt.Printf("%-20s %-10s %-10s %s\n", "HOST", "STATUS", "CONTAINERS", "VERSION")
		for _, h := range hosts {
			fmt.Printf("%-20v %-10v %-10v %v\n",
				h["id"], h["status"], h["container_count"], h["docker_version"])
		}
		return nil
	},
}

var dockerPsCmd = &cobra.Command{
	Use:   "ps <host>",
	Short: "List containers on a Docker host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		all, _ := cmd.Flags().GetBool("all")
		path := fmt.Sprintf("/api/v2/docker/hosts/%s/containers", args[0])
		if all {
			path += "?all=true"
		}

		resp, err := c.Get(path)
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		var containers []map[string]any
		json.Unmarshal(resp.Data, &containers)

		fmt.Printf("%-14s %-25s %-12s %-10s %s\n", "CONTAINER ID", "IMAGE", "STATUS", "PORTS", "NAME")
		for _, ct := range containers {
			fmt.Printf("%-14v %-25v %-12v %-10v %v\n",
				ct["id"], ct["image"], ct["status"], ct["ports"], ct["name"])
		}
		return nil
	},
}

var dockerStartCmd = &cobra.Command{
	Use:   "start <container>",
	Short: "Start a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/docker/containers/%s/start", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Container %s started\n", args[0])
		return nil
	},
}

var dockerStopCmd = &cobra.Command{
	Use:   "stop <container>",
	Short: "Stop a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/docker/containers/%s/stop", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Container %s stopped\n", args[0])
		return nil
	},
}

var dockerRestartCmd = &cobra.Command{
	Use:   "restart <container>",
	Short: "Restart a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/docker/containers/%s/restart", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Container %s restarted\n", args[0])
		return nil
	},
}

var dockerLogsCmd = &cobra.Command{
	Use:   "logs <container>",
	Short: "View container logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		tail, _ := cmd.Flags().GetInt("tail")
		path := fmt.Sprintf("/api/v2/docker/containers/%s/logs?tail=%d", args[0], tail)

		resp, err := c.Get(path)
		if err != nil {
			return err
		}

		var data map[string]any
		json.Unmarshal(resp.Data, &data)

		if logs, ok := data["logs"].(string); ok {
			fmt.Print(logs)
		}
		return nil
	},
}

func init() {
	dockerPsCmd.Flags().Bool("all", false, "Show all containers (including stopped)")
	dockerLogsCmd.Flags().Int("tail", 100, "Number of lines to show from the end of the logs")
	dockerCmd.AddCommand(dockerHostsCmd, dockerPsCmd, dockerStartCmd, dockerStopCmd, dockerRestartCmd, dockerLogsCmd)
	rootCmd.AddCommand(dockerCmd)
}
