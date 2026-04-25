package main

import (
	"fmt"
	"os"
	"strconv"

	"caspercloud/internal/apitypes"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func instancesSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "QCOW2 internal snapshots",
	}
	cmd.AddCommand(instancesSnapshotCreateCmd())
	cmd.AddCommand(instancesSnapshotListCmd())
	cmd.AddCommand(instancesSnapshotRevertCmd())
	return cmd
}

func instancesSnapshotCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "create <instance-id>",
		Short:  "Create an internal snapshot (async)",
		Args:   cobra.ExactArgs(1),
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			instID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid instance id: %w", err)
			}
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			body := apitypes.CreateSnapshotRequest{Name: name}
			var out struct {
				Data struct {
					Task *repository.Task `json:"task"`
				} `json:"data"`
			}
			if err := c.do("POST", c.projectPath("/instances/"+instID.String()+"/snapshots"), body, true, &out); err != nil {
				return err
			}
			if out.Data.Task != nil {
				fmt.Printf("Snapshot task accepted: task_id=%s\n", out.Data.Task.ID)
			}
			return nil
		},
	}
	cmd.Flags().String("name", "", "snapshot label (required)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func instancesSnapshotListCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "list <instance-id>",
		Short:  "List snapshots for an instance",
		Args:   cobra.ExactArgs(1),
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			instID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid instance id: %w", err)
			}
			var out struct {
				Data []repository.Snapshot `json:"data"`
			}
			if err := c.do("GET", c.projectPath("/instances/"+instID.String()+"/snapshots"), nil, true, &out); err != nil {
				return err
			}
			tw := tablewriter.NewWriter(os.Stdout)
			tw.SetHeader([]string{"ID", "Name", "Status", "VM was running", "Created"})
			for _, s := range out.Data {
				tw.Append([]string{
					s.ID.String(),
					s.Name,
					s.Status,
					strconv.FormatBool(s.DomainWasRunning),
					s.CreatedAt.Format("2006-01-02 15:04"),
				})
			}
			tw.Render()
			return nil
		},
	}
}

func instancesSnapshotRevertCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "revert <instance-id> <snapshot-id>",
		Short:  "Revert instance to a snapshot (async)",
		Args:   cobra.ExactArgs(2),
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			instID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid instance id: %w", err)
			}
			snapID, err := uuid.Parse(args[1])
			if err != nil {
				return fmt.Errorf("invalid snapshot id: %w", err)
			}
			var out struct {
				Data struct {
					Task *repository.Task `json:"task"`
				} `json:"data"`
			}
			path := fmt.Sprintf("/instances/%s/snapshots/%s/revert", instID.String(), snapID.String())
			if err := c.do("POST", c.projectPath(path), nil, true, &out); err != nil {
				return err
			}
			if out.Data.Task != nil {
				fmt.Printf("Revert task accepted: task_id=%s\n", out.Data.Task.ID)
			}
			return nil
		},
	}
}
