package image

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestImageFromIndexMatchesVariantPlatform(t *testing.T) {
	t.Parallel()

	target := v1.Platform{OS: "linux", Architecture: "arm64"}
	img := mustImageWithPlatform(t, v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"})
	idx := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{
		Add: img,
		Descriptor: v1.Descriptor{
			Platform: &v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"},
		},
	})

	resolved, err := imageFromIndex(idx, target)
	if err != nil {
		t.Fatalf("imageFromIndex() error = %v", err)
	}
	if resolved == nil {
		t.Fatal("imageFromIndex() returned nil image")
	}
}

func TestImageFromIndexRecursesIntoNestedIndexWithoutPlatform(t *testing.T) {
	t.Parallel()

	target := v1.Platform{OS: "linux", Architecture: "arm64"}
	img := mustImageWithPlatform(t, v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"})
	inner := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{
		Add: img,
		Descriptor: v1.Descriptor{
			Platform: &v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"},
		},
	})
	outer := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{
		Add: inner,
	})

	resolved, err := imageFromIndex(outer, target)
	if err != nil {
		t.Fatalf("imageFromIndex() error = %v", err)
	}
	if resolved == nil {
		t.Fatal("imageFromIndex() returned nil image")
	}
}

func TestImageFromIndexFallsBackToConfigPlatform(t *testing.T) {
	t.Parallel()

	target := v1.Platform{OS: "linux", Architecture: "arm64"}
	img := mustImageWithPlatform(t, v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"})
	idx := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{
		Add: img,
	})

	resolved, err := imageFromIndex(idx, target)
	if err != nil {
		t.Fatalf("imageFromIndex() error = %v", err)
	}
	if resolved == nil {
		t.Fatal("imageFromIndex() returned nil image")
	}
}

func TestImageForPlatformFallsBackFromEmptyIndex(t *testing.T) {
	t.Parallel()

	img := mustImageWithPlatform(t, v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"})
	imgDigest, err := img.Digest()
	if err != nil {
		t.Fatalf("Digest() error = %v", err)
	}
	imgManifest, err := img.RawManifest()
	if err != nil {
		t.Fatalf("RawManifest() error = %v", err)
	}
	imgMediaType, err := img.MediaType()
	if err != nil {
		t.Fatalf("MediaType() error = %v", err)
	}
	configDigest, err := img.ConfigName()
	if err != nil {
		t.Fatalf("ConfigName() error = %v", err)
	}
	configBlob, err := img.RawConfigFile()
	if err != nil {
		t.Fatalf("RawConfigFile() error = %v", err)
	}
	layers, err := img.Layers()
	if err != nil {
		t.Fatalf("Layers() error = %v", err)
	}
	layerDigest, err := layers[0].Digest()
	if err != nil {
		t.Fatalf("layer Digest() error = %v", err)
	}
	layerBlob, err := layers[0].Compressed()
	if err != nil {
		t.Fatalf("layer Compressed() error = %v", err)
	}
	defer layerBlob.Close()
	layerBytes, err := io.ReadAll(layerBlob)
	if err != nil {
		t.Fatalf("ReadAll(layer) error = %v", err)
	}

	emptyIndex := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[]}`)
	const repo = "linuxserver/emulatorjs"
	const tag = "arm64v8-latest"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case fmt.Sprintf("/v2/%s/manifests/%s", repo, tag):
			accept := r.Header.Get("Accept")
			if strings.Contains(accept, string(types.OCIImageIndex)) || strings.Contains(accept, string(types.DockerManifestList)) {
				w.Header().Set("Content-Type", string(types.OCIImageIndex))
				w.Header().Set("Docker-Content-Digest", "sha256:1111111111111111111111111111111111111111111111111111111111111111")
				_, _ = w.Write(emptyIndex)
				return
			}
			w.Header().Set("Content-Type", string(imgMediaType))
			w.Header().Set("Docker-Content-Digest", imgDigest.String())
			_, _ = w.Write(imgManifest)
		case fmt.Sprintf("/v2/%s/manifests/%s", repo, imgDigest.String()):
			w.Header().Set("Content-Type", string(imgMediaType))
			w.Header().Set("Docker-Content-Digest", imgDigest.String())
			_, _ = w.Write(imgManifest)
		case fmt.Sprintf("/v2/%s/blobs/%s", repo, configDigest.String()):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(configBlob)
		case fmt.Sprintf("/v2/%s/blobs/%s", repo, layerDigest.String()):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(layerBytes)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	ref, err := name.NewTag(fmt.Sprintf("%s/%s:%s", u.Host, repo, tag), name.WeakValidation)
	if err != nil {
		t.Fatalf("name.NewTag() error = %v", err)
	}
	desc, err := remote.Get(ref, remote.WithTransport(server.Client().Transport), remote.WithContext(context.Background()))
	if err != nil {
		t.Fatalf("remote.Get() error = %v", err)
	}

	resolved, err := imageForPlatform(context.Background(), ref, desc, v1.Platform{OS: "linux", Architecture: "arm64"})
	if err != nil {
		t.Fatalf("imageForPlatform() error = %v", err)
	}
	if resolved == nil {
		t.Fatal("imageForPlatform() returned nil image")
	}
	if _, err := resolved.ConfigFile(); err != nil {
		t.Fatalf("resolved.ConfigFile() error = %v", err)
	}
}

func mustImageWithPlatform(t *testing.T, platform v1.Platform) v1.Image {
	t.Helper()

	img, err := random.Image(1024, 1)
	if err != nil {
		t.Fatalf("random.Image() error = %v", err)
	}
	cf, err := img.ConfigFile()
	if err != nil {
		t.Fatalf("ConfigFile() error = %v", err)
	}
	cf.OS = platform.OS
	cf.Architecture = platform.Architecture
	cf.Variant = platform.Variant
	cf.OSVersion = platform.OSVersion
	cf.OSFeatures = platform.OSFeatures
	mutated, err := mutate.ConfigFile(img, cf)
	if err != nil {
		t.Fatalf("mutate.ConfigFile() error = %v", err)
	}
	return mutated
}
