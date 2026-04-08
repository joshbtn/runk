package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"runk/internal/config"
	"runk/internal/rootless"
)

type ContainerInput struct {
	Config     config.Config
	IDMap      rootless.IDMap
	ImageRef   string
	RootFS     string
	Entrypoint []string
	Cmd        []string
	Env        []string
	WorkingDir string
}

func Run(ctx context.Context, input ContainerInput) error {
	if input.IDMap.Size == 1 && !input.IDMap.UsingSubIDs && hasAptManager(input.RootFS) {
		if err := ensureAptCompatibility(input.RootFS); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "warning: rootless fallback mode detected; apt sandbox disabled for compatibility")
	}

	if input.IDMap.Strategy == rootless.StrategyProot {
		return runWithProot(ctx, input)
	}

	bundle, err := CreateBundle(BundleInput{
		Config:     input.Config,
		ImageRef:   input.ImageRef,
		RootFS:     input.RootFS,
		Entrypoint: input.Entrypoint,
		Cmd:        input.Cmd,
		Env:        input.Env,
		WorkingDir: input.WorkingDir,
		IDMap:      input.IDMap,
	})
	if err != nil {
		return err
	}
	defer func() { _ = bundle.Cleanup() }()

	stateRoot := filepath.Join(input.Config.DataRoot, "runc")
	if err := os.MkdirAll(stateRoot, 0o755); err != nil {
		return fmt.Errorf("create runc state root: %w", err)
	}

	args := []string{
		"--root", stateRoot,
		"--rootless", "true",
		"run",
		"--bundle", bundle.BundleDir,
		bundle.ID,
	}
	cmd := exec.CommandContext(ctx, input.Config.RuntimePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		_ = cleanupContainer(ctx, input.Config, bundle.ID)
		return fmt.Errorf("runc run failed: %w", err)
	}
	return nil
}

func runWithProot(ctx context.Context, input ContainerInput) error {
	args := []string{"-0", "-R", input.RootFS}

	cwd := input.WorkingDir
	if cwd == "" {
		cwd = "/"
	}
	args = append(args, "-w", cwd)

	addProotBindIfExists(&args, "/etc/resolv.conf")
	addProotBindIfExists(&args, "/etc/hosts")
	addProotBindIfExists(&args, "/etc/hostname")

	processArgs := append([]string{}, input.Entrypoint...)
	processArgs = append(processArgs, input.Cmd...)
	if len(processArgs) == 0 {
		return fmt.Errorf("empty command")
	}

	args = append(args, processArgs...)
	cmd := exec.CommandContext(ctx, "proot", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = input.Env

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("proot run failed: %w", err)
	}
	return nil
}

func addProotBindIfExists(args *[]string, path string) {
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		*args = append(*args, "-b", path+":"+path)
	}
}

func cleanupContainer(ctx context.Context, cfg config.Config, id string) error {
	stateRoot := filepath.Join(cfg.DataRoot, "runc")
	cmd := exec.CommandContext(ctx, cfg.RuntimePath, "--root", stateRoot, "--rootless", "true", "delete", "--force", id) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ensureAptCompatibility(rootfs string) error {
	aptDir := filepath.Join(rootfs, "etc", "apt", "apt.conf.d")
	if err := os.MkdirAll(aptDir, 0o755); err != nil {
		return fmt.Errorf("prepare apt compatibility directory %q: %w", aptDir, err)
	}

	aptCfgPath := filepath.Join(aptDir, "99-runk-rootless")
	content := []byte("APT::Sandbox::User \"root\";\nAcquire::Sandbox::User \"root\";\n")
	if err := os.WriteFile(aptCfgPath, content, 0o644); err != nil {
		return fmt.Errorf("write apt compatibility file %q: %w", aptCfgPath, err)
	}
	return nil
}

func hasAptManager(rootfs string) bool {
	if st, err := os.Stat(filepath.Join(rootfs, "usr", "bin", "apt-get")); err == nil && !st.IsDir() {
		return true
	}
	if st, err := os.Stat(filepath.Join(rootfs, "etc", "apt")); err == nil && st.IsDir() {
		return true
	}
	return false
}
