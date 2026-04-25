package main

import (
	"os"
	"strconv"

	"caspercloud/internal/repository"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func networksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "networks",
		Short: "Project networks",
	}
	cmd.AddCommand(networksListCmd())
	return cmd
}

func networksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "list",
		Short:  "List networks in the active project",
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			var out struct {
				Data []repository.Network `json:"data"`
			}
			if err := c.do("GET", c.projectPath("/networks"), nil, true, &out); err != nil {
				return err
			}
			tw := tablewriter.NewWriter(os.Stdout)
			tw.SetHeader([]string{"ID", "Name", "CIDR", "Gateway", "Bridge", "Default"})
			for _, n := range out.Data {
				tw.Append([]string{
					n.ID.String(),
					n.Name,
					n.CIDR,
					n.Gateway,
					n.BridgeName,
					strconv.FormatBool(n.IsDefault),
				})
			}
			tw.Render()
			return nil
		},
	}
}
