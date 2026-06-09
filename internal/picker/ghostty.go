package picker

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
	"github.com/mitchellh/go-ps"
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

// findGhosttyProcess recursively looks up the process tree starting from a
// given pid to find a process with the name "ghostty".
func findGhostty(pid int) (*os.Process, error) {
	proc, err := ps.FindProcess(pid)
	switch {
	case err != nil:
		return nil, err
	case proc.Executable() == ghosttyProcName:
		return os.FindProcess(proc.Pid())
	case proc.PPid() == 0:
		return nil, fmt.Errorf("not found")
	}
	return findGhostty(proc.PPid())
}
