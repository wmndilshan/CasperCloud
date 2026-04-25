package cloudinit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildNoCloudISO writes user-data, meta-data, optional network-config, then builds a NoCloud seed ISO at isoPath.
// cloudLocalds and genisoimage are optional explicit paths; when empty, PATH is searched.
func BuildNoCloudISO(ctx context.Context, workDir, isoPath, userData, instanceID, hostname, networkConfigYAML, cloudLocalds, genisoimage string) error {
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("mkdir workdir: %w", err)
	}
	userPath := filepath.Join(workDir, "user-data")
	metaPath := filepath.Join(workDir, "meta-data")
	if err := os.WriteFile(userPath, []byte(userData), 0o600); err != nil {
		return fmt.Errorf("write user-data: %w", err)
	}
	meta := fmt.Sprintf("instance-id: iid-%s\nlocal-hostname: %s\n", instanceID, hostname)
	if err := os.WriteFile(metaPath, []byte(meta), 0o600); err != nil {
		return fmt.Errorf("write meta-data: %w", err)
	}
	netPath := filepath.Join(workDir, "network-config")
	hasNet := strings.TrimSpace(networkConfigYAML) != ""
	if hasNet {
		if err := os.WriteFile(netPath, []byte(networkConfigYAML), 0o600); err != nil {
			return fmt.Errorf("write network-config: %w", err)
		}
	}

	localds := cloudLocalds
	if localds == "" {
		localds, _ = exec.LookPath("cloud-localds")
	}
	if localds != "" {
		args := []string{"--disk-format", "iso", isoPath, userPath, metaPath}
		if hasNet {
			args = append(args, netPath)
		}
		cmd := exec.CommandContext(ctx, localds, args...)
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("cloud-localds: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	giso := genisoimage
	if giso == "" {
		giso, _ = exec.LookPath("genisoimage")
	}
	if giso == "" {
		giso, _ = exec.LookPath("mkisofs")
	}
	if giso == "" {
		return fmt.Errorf("no ISO builder found (install cloud-image-utils for cloud-localds, or genisoimage/mkisofs)")
	}
	args := []string{
		"-output", isoPath,
		"-volid", "cidata",
		"-joliet", "-rock",
		userPath, metaPath,
	}
	if hasNet {
		args = append(args, netPath)
	}
	cmd := exec.CommandContext(ctx, giso, args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", filepath.Base(giso), err, strings.TrimSpace(string(out)))
	}
	return nil
}
