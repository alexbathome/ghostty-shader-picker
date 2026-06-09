package picker

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
)

// PreferredConfigDir returns the directory where Ghostty's config file is
// located. This is designed ot match the behaviour of Ghostty itself.
//
// See https://github.com/alexbathome/ghostty/blob/3ba5e9c24390412fb1dbb08c51008f1efdcff97b/src/config/file_load.zig#L96
func PreferredConfigDir() (string, error) {
	if runtime.GOOS == "darwin" {
		// on macOS, use ~/Library/Application Support/ghostty
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "com.mitchellh.ghostty"), nil
	}
	// everything else: use XDG_CONFIG_HOME
	return filepath.Join(xdg.ConfigHome, "ghostty"), nil
}
