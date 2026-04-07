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
