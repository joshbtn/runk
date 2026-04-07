package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"runk/internal/config"
	"runk/internal/oci"
	"runk/internal/rootless"
)

type Bundle struct {
	ID        string
	BundleDir string
}

func CreateBundle(cfg config.Config, imageRef, rootfs string, cmd []string, idMap rootless.IDMap) (Bundle, error) {
	id := newContainerID(imageRef)
	bundleDir := filepath.Join(cfg.DataRoot, "bundles", id)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return Bundle{}, err
	}

	spec, err := oci.Build(oci.RunSpecInput{
		BundleDir:   bundleDir,
		RootFSPath:  rootfs,
		ContainerID: id,
		Hostname:    "runk",
		Command:     cmd,
		NetworkMode: cfg.NetworkMode,
		IDMap:       idMap,
	})
	if err != nil {
		return Bundle{}, err
	}

	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return Bundle{}, err
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "config.json"), b, 0o644); err != nil {
		return Bundle{}, err
	}

	return Bundle{ID: id, BundleDir: bundleDir}, nil
}

func (b Bundle) Cleanup() error {
	if b.BundleDir == "" {
		return nil
	}
	return os.RemoveAll(b.BundleDir)
}

func newContainerID(imageRef string) string {
	ref := strings.NewReplacer("/", "-", ":", "-", "@", "-").Replace(imageRef)
	if len(ref) > 24 {
		ref = ref[:24]
	}
	return fmt.Sprintf("%s-%d", ref, time.Now().UnixNano())
}
