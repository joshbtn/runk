package config

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
)

type Config struct {
	DataRoot       string
	RuntimePath    string
	NetworkMode    string
	StrictRootless bool
}

func Parse(args []string) (Config, []string, error) {
	cfg := Config{}

	fs := flag.NewFlagSet("runk", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	defaultRoot, err := defaultDataRoot()
	if err != nil {
		return cfg, nil, err
	}

	fs.StringVar(&cfg.DataRoot, "data-root", defaultRoot, "runk data root")
	defaultRuntime := defaultRuntimePath()
	fs.StringVar(&cfg.RuntimePath, "runtime", defaultRuntime, "OCI runtime binary path")
	fs.StringVar(&cfg.NetworkMode, "network", "host", "network mode: host|none|slirp4netns")
	fs.BoolVar(&cfg.StrictRootless, "strict-rootless", false, "disable single-UID fallback")

	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}

	switch cfg.NetworkMode {
	case "host", "none", "slirp4netns":
	default:
		return cfg, nil, errors.New("invalid --network value (allowed: host|none|slirp4netns)")
	}

	return cfg, fs.Args(), nil
}

func defaultRuntimePath() string {
	if p := os.Getenv("RUNK_RUNTIME"); p != "" {
		return p
	}

	exe, err := os.Executable()
	if err == nil {
		sidecar := filepath.Join(filepath.Dir(exe), "runc")
		if st, statErr := os.Stat(sidecar); statErr == nil && !st.IsDir() {
			return sidecar
		}
	}

	return "runc"
}

func defaultDataRoot() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "runk"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "runk"), nil
}
