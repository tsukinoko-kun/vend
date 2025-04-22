//go:build windows

package sudo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func checkAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func Link(linkData []LinkData) error {
	if checkAdmin() {
		for _, ld := range linkData {
			if ld.New == ld.Old {
				return nil
			}
			if ld.New == "" || ld.Old == "" {
				return fmt.Errorf("invalid link data: %v", ld)
			}
			if err := os.Symlink(ld.New, ld.Old); err != nil {
				return fmt.Errorf("error creating symlink: %w", err)
			}
		}
		return nil
	}

	for i, ld := range linkData {
		changed := false
		if !filepath.IsAbs(ld.New) {
			ld.New, _ = filepath.Abs(ld.New)
			changed = true
		}
		if !filepath.IsAbs(ld.Old) {
			ld.Old, _ = filepath.Abs(ld.Old)
			changed = true
		}
		if changed {
			linkData[i] = ld
		}
	}

	// Get the path to the current executable
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %w", err)
	}

	sb := strings.Builder{}
	b64Enc := base64.NewEncoder(base64.URLEncoding, &sb)
	_ = json.NewEncoder(b64Enc).Encode(linkData)
	b64Enc.Close()

	// Prepare the arguments - properly quote to maintain separation
	args, err := syscall.UTF16PtrFromString(fmt.Sprintf(`link %s`, sb.String()))
	if err != nil {
		return fmt.Errorf("error converting arguments to UTF16: %w", err)
	}

	// Prepare the executable path and verb (runas for admin privileges)
	exePath, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("error converting executable path to UTF16: %w", err)
	}

	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return fmt.Errorf("error converting verb to UTF16: %w", err)
	}

	// Execute the command with admin privileges
	err = windows.ShellExecute(0, verb, exePath, args, nil, 1) // SW_NORMAL = 1
	if err != nil {
		return fmt.Errorf("error executing with admin privileges: %w", err)
	}

	// Wait a bit
	time.Sleep(time.Second)

	return nil
}
