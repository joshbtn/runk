package image

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Store struct {
	Root string
}

type ImageRecord struct {
	Reference      string   `json:"reference"`
	TagSafeName    string   `json:"tag_safe_name"`
	LayerDigests   []string `json:"layer_digests"`
	RootFSPath     string   `json:"rootfs_path"`
	SnapshotDriver string   `json:"snapshot_driver"`
	Env            []string `json:"env,omitempty"`
	Entrypoint     []string `json:"entrypoint,omitempty"`
	Cmd            []string `json:"cmd,omitempty"`
	WorkingDir     string   `json:"working_dir,omitempty"`
}

func NewStore(root string) (*Store, error) {
	s := &Store{Root: root}
	for _, dir := range []string{
		filepath.Join(root, "content", "blobs", "sha256"),
		filepath.Join(root, "images"),
		filepath.Join(root, "bundles"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) BlobPath(digest string) (string, error) {
	algo, encoded, ok := strings.Cut(digest, ":")
	if !ok || algo != "sha256" {
		return "", fmt.Errorf("unsupported digest %q", digest)
	}
	return filepath.Join(s.Root, "content", "blobs", "sha256", encoded), nil
}

func (s *Store) ImageDir(reference string) string {
	return filepath.Join(s.Root, "images", sanitizeRef(reference))
}

func (s *Store) SaveRecord(reference string, rec ImageRecord) error {
	imgDir := s.ImageDir(reference)
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(imgDir, "record.json"), b, 0o644)
}

func sanitizeRef(ref string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	return re.ReplaceAllString(ref, "_")
}
