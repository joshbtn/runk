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

type BundleInput struct {
	Config     config.Config
	ImageRef   string
	RootFS     string
	Entrypoint []string
	Cmd        []string
	Env        []string
	WorkingDir string
	IDMap      rootless.IDMap
}

func CreateBundle(input BundleInput) (Bundle, error) {
	id := newContainerID(input.ImageRef)
	bundleDir := filepath.Join(input.Config.DataRoot, "bundles", id)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return Bundle{}, err
	}

	spec, err := oci.Build(oci.RunSpecInput{
		BundleDir:   bundleDir,
		RootFSPath:  input.RootFS,
		ContainerID: id,
		Hostname:    "runk",
		Entrypoint:  input.Entrypoint,
		Cmd:         input.Cmd,
		Env:         input.Env,
		WorkingDir:  input.WorkingDir,
		NetworkMode: input.Config.NetworkMode,
		IDMap:       input.IDMap,
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
