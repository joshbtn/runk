package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"runk/internal/config"
	"runk/internal/image"
	"runk/internal/rootless"
	"runk/internal/runtime"
)

func main() {
	ctx := context.Background()

	cfg, args, err := config.Parse(os.Args[1:])
	if err != nil {
		exitErr(err)
	}

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "pull":
		if len(args) < 2 {
			exitErr(fmt.Errorf("usage: runk pull <image>"))
		}
		res, err := image.PullAndUnpack(ctx, cfg, args[1])
		if err != nil {
			exitErr(err)
		}
		fmt.Printf("pulled %s\n", res.Image)
		fmt.Printf("rootfs: %s\n", res.RootFS)
		fmt.Printf("snapshot: %s\n", res.SnapshotDriver)
	case "run":
		if len(args) < 2 {
			exitErr(fmt.Errorf("usage: runk run <image> -- <command> [args...]"))
		}

		preflight, err := rootless.Preflight(cfg)
		if err != nil {
			exitErr(err)
		}
		if preflight.Warning != "" {
			fmt.Fprintf(os.Stderr, "warning: %s\n", preflight.Warning)
		}

		cmd := []string{"/bin/sh"}
		if idx := indexOf(args, "--"); idx >= 0 {
			if idx+1 >= len(args) {
				exitErr(fmt.Errorf("usage: runk run <image> -- <command> [args...]"))
			}
			cmd = args[idx+1:]
			args = args[:idx]
		}

		res, err := image.PullAndUnpack(ctx, cfg, args[1])
		if err != nil {
			exitErr(err)
		}
		if err := runtime.Run(ctx, cfg, preflight.IDMap, args[1], res.RootFS, cmd); err != nil {
			exitErr(err)
		}
	default:
		exitErr(fmt.Errorf("unknown command %q", args[0]))
	}
}

func indexOf(args []string, needle string) int {
	for i, arg := range args {
		if arg == needle {
			return i
		}
	}
	return -1
}

func printUsage() {
	fmt.Println(strings.TrimSpace(`runk - daemonless, rootless OCI runner (PoC)

Usage:
  runk [global flags] pull <image>
  runk [global flags] run <image> -- <command> [args...]

Global flags:
  --data-root <path>       (default ~/.local/share/runk)
  --runtime <path>         (default runc in PATH)
  --network <mode>         host|none|slirp4netns (default host)
  --strict-rootless        fail instead of fallback when subuid/subgid is missing
`))
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
