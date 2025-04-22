//go:build !windows

package sudo

import (
	"fmt"
	"os"
)

func Link(linkData []LinkData) error {
	for _, link := range linkData {
		if err := os.Symlink(link.Old, link.New); err != nil {
			return fmt.Errorf("failed to link %s to %s: %w", link.Old, link.New, err)
		}
	}
	return nil
}
