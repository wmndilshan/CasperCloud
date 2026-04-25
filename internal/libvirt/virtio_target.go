package libvirt

import (
	"fmt"
	"regexp"
	"strings"
)

var vdTargetRE = regexp.MustCompile(`<target dev='(vd[a-z])'`)

// NextVirtioDataDiskTarget picks the next unused virtio disk target dev (vdb..vdz) from domain XML.
// The root disk is expected to be vda; attached data volumes use vdb+.
func NextVirtioDataDiskTarget(xml string) (string, error) {
	used := map[rune]struct{}{}
	for _, m := range vdTargetRE.FindAllStringSubmatch(xml, -1) {
		if len(m) < 2 || len(m[1]) != 3 {
			continue
		}
		if !strings.HasPrefix(m[1], "vd") {
			continue
		}
		letter := rune(m[1][2])
		used[letter] = struct{}{}
	}
	for r := 'b'; r <= 'z'; r++ {
		if _, ok := used[r]; !ok {
			return fmt.Sprintf("vd%c", r), nil
		}
	}
	return "", fmt.Errorf("no free virtio disk slot (vdb–vdz exhausted)")
}
