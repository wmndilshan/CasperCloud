package cloudinit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildNoCloudISO writes user-data and meta-data then builds a NoCloud seed ISO at isoPath.
// cloudLocalds and genisoimage are optional explicit paths; when empty, the first available
// tool in PATH is used (cloud-localds preferred, then genisoimage).
func BuildNoCloudISO(ctx context.Context, workDir, isoPath, userData, instanceID, hostname, cloudLocalds, genisoimage string) error {
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

	localds := cloudLocalds
	if localds == "" {
		localds, _ = exec.LookPath("cloud-localds")
	}
	if localds != "" {
		cmd := exec.CommandContext(ctx, localds, "--disk-format", "iso", isoPath, userPath, metaPath)
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
	cmd := exec.CommandContext(ctx, giso,
		"-output", isoPath,
		"-volid", "cidata",
		"-joliet", "-rock",
		userPath, metaPath,
	)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", filepath.Base(giso), err, strings.TrimSpace(string(out)))
	}
	return nil
}
