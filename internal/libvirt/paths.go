package libvirt

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ImagePathFromSource returns a local filesystem path for a base disk image.
// Accepts plain paths or file:// URLs; rejects other schemes (e.g. http).
func ImagePathFromSource(sourceURL string) (string, error) {
	raw := strings.TrimSpace(sourceURL)
	if raw == "" {
		return "", fmt.Errorf("empty image source")
	}
	path := raw
	if u, err := url.Parse(raw); err == nil && u.Scheme != "" {
		switch strings.ToLower(u.Scheme) {
		case "file":
			path = u.Path
			if path == "" {
				path = u.Opaque
			}
		case "http", "https":
			return "", fmt.Errorf("remote image URL is not supported for VM provisioning: %q", sourceURL)
		default:
			return "", fmt.Errorf("unsupported image URL scheme %q", u.Scheme)
		}
	}
	if strings.Contains(path, "://") {
		return "", fmt.Errorf("unsupported image reference: %q", sourceURL)
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve image path: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("base image not accessible at %q: %w", abs, err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("base image path is a directory: %q", abs)
	}
	return abs, nil
}
