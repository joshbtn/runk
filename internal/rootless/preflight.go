package rootless

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"runk/internal/config"
)

type PreflightResult struct {
	IDMap   IDMap
	Warning string
}

func Preflight(cfg config.Config) (PreflightResult, error) {
	if _, err := exec.LookPath(cfg.RuntimePath); err != nil {
		return PreflightResult{}, fmt.Errorf("runtime not found (%s): %w (hint: run 'make runc-install' to provision sidecar)", cfg.RuntimePath, err)
	}

	if runtime.GOOS != "linux" {
		return PreflightResult{}, fmt.Errorf("runk PoC currently supports linux hosts only")
	}

	if err := checkUserNS(); err != nil {
		return PreflightResult{}, err
	}

	idMap, warning, err := ResolveIDMap(cfg.StrictRootless)
	if err != nil {
		return PreflightResult{}, err
	}

	if cfg.NetworkMode == "slirp4netns" {
		if _, err := exec.LookPath("slirp4netns"); err != nil {
			return PreflightResult{}, fmt.Errorf("--network=slirp4netns requested but slirp4netns binary not found")
		}
	}

	return PreflightResult{IDMap: idMap, Warning: warning}, nil
}

func checkUserNS() error {
	data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	if err == nil {
		if strings.TrimSpace(string(data)) == "0" {
			return fmt.Errorf("kernel blocks unprivileged user namespaces (kernel.unprivileged_userns_clone=0)")
		}
	}

	data, err = os.ReadFile("/proc/sys/user/max_user_namespaces")
	if err == nil {
		if strings.TrimSpace(string(data)) == "0" {
			return fmt.Errorf("kernel user namespaces disabled (user.max_user_namespaces=0)")
		}
	}
	return nil
}
