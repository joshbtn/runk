package image

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
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
