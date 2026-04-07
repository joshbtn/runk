package image

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ApplyLayerTar(layer io.Reader, rootfs string) error {
	tr := tar.NewReader(layer)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		rel := filepath.Clean(hdr.Name)
		if rel == "." || rel == "" {
			continue
		}
		target, err := safeJoin(rootfs, rel)
		if err != nil {
			return err
		}

		base := filepath.Base(rel)
		if strings.HasPrefix(base, ".wh.") {
			if base == ".wh..wh..opq" {
				dir := filepath.Dir(target)
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					if rmErr := os.RemoveAll(filepath.Join(dir, e.Name())); rmErr != nil {
						return rmErr
					}
				}
				continue
			}
			original := strings.TrimPrefix(base, ".wh.")
			toRemove := filepath.Join(filepath.Dir(target), original)
			if err := os.RemoveAll(toRemove); err != nil {
				return err
			}
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		case tar.TypeLink:
			linkTarget, err := safeJoin(rootfs, hdr.Linkname)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}
		default:
			// Skip unsupported special nodes in PoC.
		}
	}
}

func safeJoin(root, rel string) (string, error) {
	joined := filepath.Join(root, rel)
	cleanRoot := filepath.Clean(root)
	cleanJoined := filepath.Clean(joined)
	if cleanJoined != cleanRoot && !strings.HasPrefix(cleanJoined, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path outside rootfs: %q", rel)
	}
	return cleanJoined, nil
}
