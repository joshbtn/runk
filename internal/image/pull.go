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
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithPlatform(platform))
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
