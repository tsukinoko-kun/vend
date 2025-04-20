package user

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

var Current, _ = user.Current()

func Location() string {
	if xdgData, ok := os.LookupEnv("XDG_DATA_HOME"); ok {
		return filepath.Join(xdgData, "vend")
	} else {
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(Current.HomeDir, "Library", "Application Support", "vend")
		case "linux":
			return filepath.Join(Current.HomeDir, ".local", "share", "vend")
		case "windows":
			return filepath.Join(os.Getenv("APPDATA"), "vend")
		default:
			return filepath.Join(Current.HomeDir, ".vend")
		}
	}
}
