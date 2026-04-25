package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"caspercloud/internal/apitypes"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func instancesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instances",
		Short: "Virtual machines",
	}
	cmd.AddCommand(instancesCreateCmd())
	cmd.AddCommand(instancesListCmd())
	cmd.AddCommand(instancesSSHCmd())
	return cmd
}

func requireProjectPreRun(_ *cobra.Command, _ []string) {
	_ = requireProjectAuth()
}

func instancesListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:    "list",
		Short:  "List instances in the active project",
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			var out struct {
				Data []repository.Instance `json:"data"`
			}
			if err := c.do("GET", c.projectPath("/instances"), nil, true, &out); err != nil {
				return err
			}
			tw := tablewriter.NewWriter(os.Stdout)
			tw.SetHeader([]string{"ID", "Name", "State", "Image", "IPv4", "Created"})
			for _, in := range out.Data {
				ip := ""
				if in.IPv4Address != nil {
					ip = *in.IPv4Address
				}
				tw.Append([]string{
					in.ID.String(),
					in.Name,
					in.State,
					in.ImageID.String(),
					ip,
					in.CreatedAt.Format("2006-01-02 15:04"),
				})
			}
			tw.Render()
			return nil
		},
	}
	return c
}

func instancesCreateCmd() *cobra.Command {
	var (
		memoryMiB int
		vcpus     int
		hostname  string
		username  string
		sshKey    string
		netID     string
	)
	cmd := &cobra.Command{
		Use:    "create",
		Short:  "Create an instance and wait until it is running",
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			name, _ := cmd.Flags().GetString("name")
			imageRef, _ := cmd.Flags().GetString("image")
			if strings.TrimSpace(name) == "" || strings.TrimSpace(imageRef) == "" {
				return errors.New("--name and --image are required")
			}
			if memoryMiB > 0 || vcpus > 0 {
				printNote("Note: --memory and --vcpus are not exposed by the API yet; the server uses its configured defaults.")
			}

			imageID, err := resolveImageID(c, imageRef)
			if err != nil {
				return err
			}
			host := hostname
			if strings.TrimSpace(host) == "" {
				host = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "_", "-"))
			}
			user := username
			if strings.TrimSpace(user) == "" {
				user = "ubuntu"
			}
			key, err := resolveSSHPublicKey(sshKey)
			if err != nil {
				return err
			}

			req := apitypes.CreateInstanceRequest{
				Name:         name,
				ImageID:      imageID.String(),
				Hostname:     host,
				Username:     user,
				SSHPublicKey: key,
			}
			if strings.TrimSpace(netID) != "" {
				req.NetworkID = netID
			}

			var accepted struct {
				Data struct {
					Instance *repository.Instance `json:"instance"`
					Task     *repository.Task     `json:"task"`
				} `json:"data"`
			}
			if err := c.do("POST", c.projectPath("/instances"), req, true, &accepted); err != nil {
				return err
			}
			inst := accepted.Data.Instance
			if inst == nil {
				return errors.New("API returned no instance")
			}
			fmt.Fprintf(os.Stderr, "Instance %s (%s) accepted; waiting for running state…\n", inst.Name, inst.ID)

			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Minute)
			defer cancel()
			for {
				var one struct {
					Data repository.Instance `json:"data"`
				}
				if err := c.do("GET", c.projectPath("/instances/"+inst.ID.String()), nil, true, &one); err != nil {
					return err
				}
				switch one.Data.State {
				case "running":
					fmt.Printf("Instance %s is running\n", one.Data.ID)
					return nil
				case "error":
					return fmt.Errorf("instance entered error state")
				default:
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("timeout waiting for instance to run (last state %q)", one.Data.State)
				case <-time.After(2 * time.Second):
				}
			}
		},
	}
	cmd.Flags().String("name", "", "instance name (required)")
	cmd.Flags().String("image", "", "image UUID or name (required)")
	cmd.Flags().IntVar(&memoryMiB, "memory", 0, "memory in MiB (not sent to API yet)")
	cmd.Flags().IntVar(&vcpus, "vcpus", 0, "vCPUs (not sent to API yet)")
	cmd.Flags().StringVar(&hostname, "hostname", "", "cloud-init hostname (default derived from --name)")
	cmd.Flags().StringVar(&username, "username", "", "login user for cloud-init (default: ubuntu)")
	cmd.Flags().StringVar(&sshKey, "ssh-public-key", "", "SSH public key string (default: read ~/.ssh/id_ed25519.pub or id_rsa.pub)")
	cmd.Flags().StringVar(&netID, "network-id", "", "optional network UUID")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("image")
	return cmd
}

func instancesSSHCmd() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:    "ssh <instance-id> [--] [ssh-args...]",
		Short:  "SSH to an instance using its API-reported IPv4 address",
		Args:   cobra.MinimumNArgs(1),
		PreRun: requireProjectPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := requireProjectAuth()
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid instance id: %w", err)
			}
			sshExtra := args[1:]
			if user == "" {
				user = os.Getenv("USER")
			}
			if user == "" {
				return errors.New("set --user or USER for ssh login name")
			}

			var one struct {
				Data repository.Instance `json:"data"`
			}
			if err := c.do("GET", c.projectPath("/instances/"+id.String()), nil, true, &one); err != nil {
				return err
			}
			if one.Data.IPv4Address == nil || strings.TrimSpace(*one.Data.IPv4Address) == "" {
				return fmt.Errorf("instance %s has no IPv4 yet (state=%s)", id, one.Data.State)
			}
			host := *one.Data.IPv4Address
			target := user + "@" + host
			argv := append([]string{target}, sshExtra...)
			xc := exec.Command("ssh", argv...)
			xc.Stdin = os.Stdin
			xc.Stdout = os.Stdout
			xc.Stderr = os.Stderr
			return xc.Run()
		},
	}
	cmd.Flags().StringVar(&user, "user", "", "SSH remote user (default: $USER)")
	return cmd
}

func resolveImageID(c *Client, nameOrID string) (uuid.UUID, error) {
	nameOrID = strings.TrimSpace(nameOrID)
	if id, err := uuid.Parse(nameOrID); err == nil {
		return id, nil
	}
	var list struct {
		Data []repository.Image `json:"data"`
	}
	if err := c.do("GET", c.projectPath("/images"), nil, true, &list); err != nil {
		return uuid.Nil, err
	}
	for _, im := range list.Data {
		if strings.EqualFold(im.Name, nameOrID) {
			return im.ID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("no image matches %q (use UUID or exact catalog name)", nameOrID)
}

func resolveSSHPublicKey(flagVal string) (string, error) {
	if s := strings.TrimSpace(flagVal); s != "" && !strings.HasPrefix(s, "@") {
		return s, nil
	}
	// @file path convention
	if strings.HasPrefix(strings.TrimSpace(flagVal), "@") {
		b, err := os.ReadFile(strings.TrimPrefix(strings.TrimSpace(flagVal), "@"))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, name := range []string{".ssh/id_ed25519.pub", ".ssh/id_rsa.pub"} {
		b, err := os.ReadFile(filepath.Join(home, name))
		if err == nil {
			return strings.TrimSpace(string(b)), nil
		}
	}
	return "", errors.New("SSH public key not set: use --ssh-public-key or ensure ~/.ssh/id_ed25519.pub exists")
}
