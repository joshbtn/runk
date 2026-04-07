package snapshot

import (
	"os"
	"runtime"
	"strings"
)

const (
	DriverOverlay = "overlayfs"
	DriverFuse    = "fuse-overlayfs"
	DriverCopy    = "copy"
)

func SelectDriver() string {
	if runtime.GOOS != "linux" {
		return DriverCopy
	}

	if hasFilesystem("overlay") {
		return DriverOverlay
	}
	if _, err := os.Stat("/dev/fuse"); err == nil {
		return DriverFuse
	}
	return DriverCopy
}

func hasFilesystem(name string) bool {
	b, err := os.ReadFile("/proc/filesystems")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, name) {
			return true
		}
	}
	return false
}
