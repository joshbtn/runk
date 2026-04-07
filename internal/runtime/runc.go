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

func Run(ctx context.Context, cfg config.Config, idMap rootless.IDMap, imageRef, rootfs string, command []string) error {
	if idMap.Size == 1 && !idMap.UsingSubIDs && hasAptManager(rootfs) {
		if err := ensureAptCompatibility(rootfs); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "warning: single-ID rootless mode detected; apt sandbox disabled for compatibility")
	}

	bundle, err := CreateBundle(cfg, imageRef, rootfs, command, idMap)
	if err != nil {
		return err
	}
	defer func() { _ = bundle.Cleanup() }()

	stateRoot := filepath.Join(cfg.DataRoot, "runc")
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
	cmd := exec.CommandContext(ctx, cfg.RuntimePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		_ = cleanupContainer(ctx, cfg, bundle.ID)
		return fmt.Errorf("runc run failed: %w", err)
	}
	return nil
}

func cleanupContainer(ctx context.Context, cfg config.Config, id string) error {
	stateRoot := filepath.Join(cfg.DataRoot, "runc")
	cmd := exec.CommandContext(ctx, cfg.RuntimePath, "--root", stateRoot, "--rootless", "true", "delete", "--force", id)
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
