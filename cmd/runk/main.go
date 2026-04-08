package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
		runFlags := flag.NewFlagSet("run", flag.ContinueOnError)
		runFlags.SetOutput(os.Stderr)
		var envFlags []string
		var entrypointFlag string
		runFlags.Func("env", "set environment variable `KEY=VALUE` (repeatable)", func(s string) error {
			if !strings.Contains(s, "=") {
				return fmt.Errorf("--env value must be KEY=VALUE, got %q", s)
			}
			envFlags = append(envFlags, s)
			return nil
		})
		runFlags.StringVar(&entrypointFlag, "entrypoint", "", "override image entrypoint")
		if err := runFlags.Parse(args[1:]); err != nil {
			os.Exit(1)
		}
		runArgs := runFlags.Args()
		if len(runArgs) < 1 {
			exitErr(fmt.Errorf("usage: runk run [--env KEY=VALUE] [--entrypoint CMD] <image> [-- <command> [args...]]"))
		}

		preflight, err := rootless.Preflight(cfg)
		if err != nil {
			exitErr(err)
		}
		if preflight.Warning != "" {
			fmt.Fprintf(os.Stderr, "warning: %s\n", preflight.Warning)
		}

		imageRef := runArgs[0]
		var userCmd []string
		if idx := indexOf(runArgs, "--"); idx >= 0 && idx+1 < len(runArgs) {
			userCmd = runArgs[idx+1:]
		}

		res, err := image.PullAndUnpack(ctx, cfg, imageRef)
		if err != nil {
			exitErr(err)
		}

		resolvedEntrypoint := res.Entrypoint
		if entrypointFlag != "" {
			resolvedEntrypoint = strings.Fields(entrypointFlag)
		}
		resolvedCmd := res.Cmd
		if len(userCmd) > 0 {
			resolvedCmd = userCmd
		}
		if len(resolvedEntrypoint) == 0 && len(resolvedCmd) == 0 {
			resolvedCmd = defaultInteractiveCommand(res.RootFS)
		}

		if err := runtime.Run(ctx, runtime.ContainerInput{
			Config:     cfg,
			IDMap:      preflight.IDMap,
			ImageRef:   imageRef,
			RootFS:     res.RootFS,
			Entrypoint: resolvedEntrypoint,
			Cmd:        resolvedCmd,
			Env:        mergeEnv(res.Env, envFlags),
			WorkingDir: res.WorkingDir,
		}); err != nil {
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
  runk [global flags] run [run flags] <image> [-- <command> [args...]]

Global flags:
  --data-root <path>       (default ~/.local/share/runk)
  --runtime <path>         (default runc in PATH)
  --network <mode>         host|none|slirp4netns (default host)
	--strict-rootless        fail instead of fallback when subuid/subgid is missing
	--single-user-fallback   use legacy single-user fallback instead of default proot fallback

Run flags:
  --env KEY=VALUE          set or override an environment variable (repeatable)
  --entrypoint <cmd>       override the image entrypoint
`))
}

func defaultInteractiveCommand(rootfs string) []string {
	bashPath := filepath.Join(rootfs, "bin", "bash")
	if st, err := os.Stat(bashPath); err == nil && !st.IsDir() {
		return []string{"/bin/bash", "-i"}
	}
	return []string{"/bin/sh", "-i"}
}

// mergeEnv merges image env vars with CLI overrides.
// Image env is the base; overrides replace an existing key or append new ones.
func mergeEnv(base, overrides []string) []string {
	m := make(map[string]int, len(base))
	result := make([]string, len(base))
	copy(result, base)
	for i, e := range base {
		k, _, _ := strings.Cut(e, "=")
		m[k] = i
	}
	for _, e := range overrides {
		k, _, _ := strings.Cut(e, "=")
		if idx, exists := m[k]; exists {
			result[idx] = e
		} else {
			m[k] = len(result)
			result = append(result, e)
		}
	}
	return result
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
