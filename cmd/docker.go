package cmd

import (
	"fmt"

	"github.com/labtether/labtether-cli/internal/client"
	"github.com/spf13/cobra"
)

const (
	minDockerLogTailLines = 1
	maxDockerLogTailLines = 10000
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

		return printList(resp, "hosts",
			col("HOST", "agent_id"), col("OS", "engine_os"),
			col("CONTAINERS", "container_count"), col("VERSION", "engine_version"))
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

		path := fmt.Sprintf("/api/v2/docker/hosts/%s/containers", pathSegment(args[0]))

		resp, err := c.Get(path)
		if err != nil {
			return err
		}

		return printList(resp, "containers",
			colFormat("CONTAINER ID", "id", shortID), col("IMAGE", "image"), col("STATUS", "status"),
			col("PORTS", "ports"), col("NAME", "name"))
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
		resp, err := requestDockerContainerAction(c, args[0], "start")
		if err != nil {
			return err
		}
		printMutationResult(resp, "Container %s started\n", args[0])
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
		resp, err := requestDockerContainerAction(c, args[0], "stop")
		if err != nil {
			return err
		}
		printMutationResult(resp, "Container %s stopped\n", args[0])
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
		resp, err := requestDockerContainerAction(c, args[0], "restart")
		if err != nil {
			return err
		}
		printMutationResult(resp, "Container %s restarted\n", args[0])
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
		path := fmt.Sprintf("/api/v2/docker/containers/%s/logs?tail=%d", pathSegment(args[0]), tail)

		resp, err := c.Get(path)
		if err != nil {
			return err
		}

		var data map[string]any
		if err := decodeResponseData(resp, &data); err != nil {
			return err
		}

		if logs, ok := data["logs"].(string); ok {
			fmt.Fprint(cmd.OutOrStdout(), logs)
		}
		return nil
	},
}

func init() {
	dockerPsCmd.Flags().Bool("all", false, "Accepted for compatibility; Hub already returns all containers")
	dockerLogsCmd.Flags().Var(
		newBoundedIntValue(100, "tail", "", minDockerLogTailLines, maxDockerLogTailLines),
		"tail",
		"Number of lines to show from the end of the logs",
	)
	dockerCmd.AddCommand(dockerHostsCmd, dockerPsCmd, dockerStartCmd, dockerStopCmd, dockerRestartCmd, dockerLogsCmd)
	rootCmd.AddCommand(dockerCmd)
}

func requestDockerContainerAction(c *client.Client, containerID, verb string) (*client.V2Response, error) {
	resp, err := c.Post(fmt.Sprintf("/api/v2/docker/containers/%s/action", pathSegment(containerID)),
		map[string]string{"action": "container." + verb})
	if err != nil {
		return nil, err
	}
	var data struct {
		Result struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"result"`
	}
	if err := decodeResponseData(resp, &data); err != nil {
		return nil, err
	}
	if data.Result.Status != "succeeded" {
		if data.Result.Message != "" {
			return nil, fmt.Errorf("Docker action failed: %s", data.Result.Message)
		}
		return nil, fmt.Errorf("Docker action failed with status %q", data.Result.Status)
	}
	return resp, nil
}
