package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

// VolumeQCOW2Path returns the canonical path for a project's volume disk image.
func VolumeQCOW2Path(volumesDir string, volumeID uuid.UUID) string {
	return filepath.Join(volumesDir, volumeID.String()+".qcow2")
}

// CreateSparseQCOW2 runs `qemu-img create -f qcow2` so the file is sparse until written.
func CreateSparseQCOW2(ctx context.Context, qemuImgPath, destPath string, sizeGB int) error {
	if sizeGB <= 0 {
		return fmt.Errorf("size_gb must be positive")
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("mkdir volumes parent: %w", err)
	}
	sizeArg := fmt.Sprintf("%dG", sizeGB)
	cmd := exec.CommandContext(ctx, qemuImgPath, "create", "-f", "qcow2", destPath, sizeArg)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("qemu-img create: %w", err)
	}
	return nil
}
