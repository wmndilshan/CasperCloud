package main

import (
	"os"
	"strconv"

	"caspercloud/internal/repository"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func volumesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volumes",
		Short: "Manage block volumes",
	}
	cmd.AddCommand(volumesListCmd())
	return cmd
}

func volumesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "list",
		Short:  "List volumes in the active project",
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			var out struct {
				Data []repository.Volume `json:"data"`
			}
			if err := c.do("GET", c.projectPath("/volumes"), nil, true, &out); err != nil {
				return err
			}
			tw := tablewriter.NewWriter(os.Stdout)
			tw.SetHeader([]string{"ID", "Name", "Size (GB)", "Status", "Created"})
			for _, v := range out.Data {
				tw.Append([]string{
					v.ID.String(),
					v.Name,
					strconv.Itoa(v.SizeGB),
					v.Status,
					v.CreatedAt.Format("2006-01-02 15:04"),
				})
			}
			tw.Render()
			return nil
		},
	}
}
