package image

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"runk/internal/config"
	"runk/internal/snapshot"
)

type PullResult struct {
	Image          string
	RootFS         string
	SnapshotDriver string
	Env            []string
	Entrypoint     []string
	Cmd            []string
	WorkingDir     string
}

func PullAndUnpack(ctx context.Context, cfg config.Config, imageRef string) (PullResult, error) {
	store, err := NewStore(cfg.DataRoot)
	if err != nil {
		return PullResult{}, err
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return PullResult{}, fmt.Errorf("parse image reference: %w", err)
	}

	platform := v1.Platform{
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}
	desc, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return PullResult{}, fmt.Errorf("pull image: %w", err)
	}
	img, err := imageForPlatform(desc, platform)
	if err != nil {
		return PullResult{}, fmt.Errorf("pull image: %w", err)
	}

	imgDir := store.ImageDir(imageRef)
	rootfs := filepath.Join(imgDir, "rootfs")
	if err := os.RemoveAll(rootfs); err != nil {
		return PullResult{}, err
	}
	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return PullResult{}, err
	}

	layers, err := img.Layers()
	if err != nil {
		return PullResult{}, err
	}

	digests := make([]string, 0, len(layers))
	for _, layer := range layers {
		dgst, err := layer.Digest()
		if err != nil {
			return PullResult{}, err
		}
		digests = append(digests, dgst.String())

		blobPath, err := store.BlobPath(dgst.String())
		if err != nil {
			return PullResult{}, err
		}
		if _, err := os.Stat(blobPath); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(blobPath), 0o755); err != nil {
				return PullResult{}, err
			}
			rc, err := layer.Compressed()
			if err != nil {
				return PullResult{}, err
			}
			f, err := os.Create(blobPath)
			if err != nil {
				_ = rc.Close()
				return PullResult{}, err
			}
			if _, err := io.Copy(f, rc); err != nil {
				_ = rc.Close()
				_ = f.Close()
				return PullResult{}, err
			}
			_ = rc.Close()
			if err := f.Close(); err != nil {
				return PullResult{}, err
			}
		}

		rc, err := os.Open(blobPath)
		if err != nil {
			return PullResult{}, err
		}
		gz, err := gzip.NewReader(rc)
		if err != nil {
			_ = rc.Close()
			return PullResult{}, err
		}
		if err := ApplyLayerTar(gz, rootfs); err != nil {
			_ = gz.Close()
			_ = rc.Close()
			return PullResult{}, err
		}
		_ = gz.Close()
		_ = rc.Close()
	}

	driver := snapshot.SelectDriver()
	cfgFile, err := img.ConfigFile()
	if err != nil {
		return PullResult{}, fmt.Errorf("image config: %w", err)
	}
	rec := ImageRecord{
		Reference:      imageRef,
		TagSafeName:    filepath.Base(imgDir),
		LayerDigests:   digests,
		RootFSPath:     rootfs,
		SnapshotDriver: driver,
		Env:            cfgFile.Config.Env,
		Entrypoint:     cfgFile.Config.Entrypoint,
		Cmd:            cfgFile.Config.Cmd,
		WorkingDir:     cfgFile.Config.WorkingDir,
	}
	if err := store.SaveRecord(imageRef, rec); err != nil {
		return PullResult{}, err
	}

	return PullResult{
		Image:          imageRef,
		RootFS:         rootfs,
		SnapshotDriver: driver,
		Env:            rec.Env,
		Entrypoint:     rec.Entrypoint,
		Cmd:            rec.Cmd,
		WorkingDir:     rec.WorkingDir,
	}, nil
}

func imageForPlatform(desc *remote.Descriptor, platform v1.Platform) (v1.Image, error) {
	if desc.MediaType.IsImage() {
		return desc.Image()
	}
	if !desc.MediaType.IsIndex() {
		return nil, fmt.Errorf("unsupported manifest media type %s", desc.MediaType)
	}

	idx, err := desc.ImageIndex()
	if err != nil {
		return nil, err
	}

	img, err := imageFromIndex(idx, platform)
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, fmt.Errorf("no child with platform %s in index", platform.String())
	}
	return img, nil
}

func imageFromIndex(idx v1.ImageIndex, platform v1.Platform) (v1.Image, error) {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, err
	}

	for _, child := range manifest.Manifests {
		if child.Platform != nil && child.Platform.Satisfies(platform) {
			img, err := imageFromChild(idx, child, platform)
			if err != nil {
				return nil, err
			}
			if img != nil {
				return img, nil
			}
		}
	}

	for _, child := range manifest.Manifests {
		if child.Platform == nil || child.MediaType.IsIndex() {
			img, err := imageFromChild(idx, child, platform)
			if err != nil {
				return nil, err
			}
			if img != nil {
				return img, nil
			}
		}
	}

	return nil, nil
}

func imageFromChild(idx v1.ImageIndex, child v1.Descriptor, platform v1.Platform) (v1.Image, error) {
	switch {
	case child.MediaType.IsIndex():
		nested, err := idx.ImageIndex(child.Digest)
		if err != nil {
			return nil, err
		}
		return imageFromIndex(nested, platform)
	case child.MediaType.IsImage(), child.MediaType == types.MediaType(""):
		img, err := idx.Image(child.Digest)
		if err != nil {
			return nil, err
		}
		if child.Platform != nil && child.Platform.Satisfies(platform) {
			return img, nil
		}
		cfg, err := img.ConfigFile()
		if err != nil {
			return nil, err
		}
		if cfgPlatform := cfg.Platform(); cfgPlatform != nil && cfgPlatform.Satisfies(platform) {
			return img, nil
		}
		return nil, nil
	default:
		return nil, nil
	}
}
