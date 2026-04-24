package libvirt

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CloneQCOW2 creates an independent qcow2 copy at dst from backing src using qemu-img convert.
func CloneQCOW2(ctx context.Context, qemuImgBin, src, dst string) error {
	if qemuImgBin == "" {
		qemuImgBin = "qemu-img"
	}
	cmd := exec.CommandContext(ctx, qemuImgBin, "convert", "-p", "-O", "qcow2", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img convert: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
