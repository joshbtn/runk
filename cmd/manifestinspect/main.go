package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	remotetransport "github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type node struct {
	Kind          string            `json:"kind"`
	MediaType     string            `json:"mediaType"`
	Digest        string            `json:"digest,omitempty"`
	Platform      string            `json:"platform,omitempty"`
	Size          int64             `json:"size,omitempty"`
	ArtifactType  string            `json:"artifactType,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Reference     string            `json:"reference,omitempty"`
	Children      []node            `json:"children,omitempty"`
	ConfigOS      string            `json:"configOS,omitempty"`
	ConfigArch    string            `json:"configArch,omitempty"`
	ConfigVariant string            `json:"configVariant,omitempty"`
	ConfigDigest  string            `json:"configDigest,omitempty"`
	Error         string            `json:"error,omitempty"`
}

type report struct {
	Reference          string `json:"reference"`
	TargetPlatform     string `json:"targetPlatform"`
	DefaultGet         node   `json:"defaultGet"`
	ImageOnlyManifest  node   `json:"imageOnlyManifest"`
	IndexOnlyManifest  node   `json:"indexOnlyManifest"`
	ModelCaptureTODO   string `json:"modelCaptureTODO"`
	ResolutionHintTODO string `json:"resolutionHintTODO"`
}

func main() {
	var targetPlatform string
	flag.StringVar(&targetPlatform, "platform", "linux/arm64", "platform hint used for analysis output")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/manifestinspect --platform linux/arm64 <image-ref>")
		os.Exit(2)
	}

	ctx := context.Background()
	ref, err := name.ParseReference(flag.Arg(0), name.WeakValidation)
	if err != nil {
		fail(err)
	}

	rep := report{
		Reference:      ref.String(),
		TargetPlatform: targetPlatform,
		ModelCaptureTODO: "TODO(model): extend persistent image metadata model to capture descriptor-level fields " +
			"(media type, digest, size, artifactType, annotations, platform, nested index lineage).",
		ResolutionHintTODO: "TODO(pull): consider trying both index-first and image-only manifest negotiation when registry responses are inconsistent.",
	}

	desc, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(ctx))
	if err != nil {
		rep.DefaultGet = node{Kind: "error", Error: err.Error(), Reference: ref.String()}
	} else {
		rep.DefaultGet = describeDescriptor(ctx, ref, desc)
	}

	rep.ImageOnlyManifest = fetchRawManifest(ctx, ref, []types.MediaType{types.DockerManifestSchema2, types.OCIManifestSchema1})
	rep.IndexOnlyManifest = fetchRawManifest(ctx, ref, []types.MediaType{types.DockerManifestList, types.OCIImageIndex})

	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(out))
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func describeDescriptor(ctx context.Context, ref name.Reference, desc *remote.Descriptor) node {
	n := node{
		Reference:    ref.String(),
		MediaType:    string(desc.MediaType),
		Digest:       desc.Digest.String(),
		Size:         desc.Size,
		ArtifactType: desc.ArtifactType,
		Annotations:  desc.Annotations,
	}
	if desc.Platform != nil {
		n.Platform = desc.Platform.String()
	}

	switch {
	case desc.MediaType.IsIndex():
		n.Kind = "index"
		idx, err := desc.ImageIndex()
		if err != nil {
			n.Error = err.Error()
			return n
		}
		m, err := idx.IndexManifest()
		if err != nil {
			n.Error = err.Error()
			return n
		}
		for _, child := range m.Manifests {
			childRef := ref.Context().Digest(child.Digest.String())
			childDesc, err := remote.Get(childRef, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(ctx))
			if err != nil {
				n.Children = append(n.Children, node{
					Kind:      "error",
					Digest:    child.Digest.String(),
					MediaType: string(child.MediaType),
					Error:     err.Error(),
				})
				continue
			}
			cn := describeDescriptor(ctx, childRef, childDesc)
			if cn.Platform == "" && child.Platform != nil {
				cn.Platform = child.Platform.String()
			}
			cn.Annotations = mergeAnnotations(cn.Annotations, child.Annotations)
			n.Children = append(n.Children, cn)
		}
	case desc.MediaType.IsImage() || desc.MediaType == "":
		n.Kind = "image"
		img, err := desc.Image()
		if err != nil {
			n.Error = err.Error()
			return n
		}
		cfg, err := img.ConfigFile()
		if err != nil {
			n.Error = err.Error()
			return n
		}
		n.ConfigOS = cfg.OS
		n.ConfigArch = cfg.Architecture
		n.ConfigVariant = cfg.Variant
		if d, err := img.ConfigName(); err == nil {
			n.ConfigDigest = d.String()
		}
	default:
		n.Kind = "unknown"
	}

	return n
}

func mergeAnnotations(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if _, exists := out[k]; !exists {
			out[k] = v
		}
	}
	return out
}

func fetchRawManifest(ctx context.Context, ref name.Reference, accept []types.MediaType) node {
	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return node{Kind: "error", Error: err.Error()}
	}
	rt, err := remotetransport.NewWithContext(ctx, ref.Context().Registry, auth, remote.DefaultTransport, []string{ref.Context().Scope(remotetransport.PullScope)})
	if err != nil {
		return node{Kind: "error", Error: err.Error()}
	}

	u := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", ref.Context().Scheme(), ref.Context().RegistryStr(), ref.Context().RepositoryStr(), ref.Identifier())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return node{Kind: "error", Error: err.Error()}
	}
	parts := make([]string, 0, len(accept))
	for _, mt := range accept {
		parts = append(parts, string(mt))
	}
	req.Header.Set("Accept", strings.Join(parts, ","))

	resp, err := (&http.Client{Transport: rt}).Do(req)
	if err != nil {
		return node{Kind: "error", Error: err.Error()}
	}
	defer resp.Body.Close()
	if err := remotetransport.CheckError(resp, http.StatusOK); err != nil {
		return node{Kind: "error", Error: err.Error()}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return node{Kind: "error", Error: err.Error()}
	}
	d, _, err := v1.SHA256(bytes.NewReader(body))
	if err != nil {
		return node{Kind: "error", Error: err.Error()}
	}
	mt := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	n := node{
		Kind:      "raw-manifest",
		MediaType: mt,
		Digest:    d.String(),
		Size:      int64(len(body)),
	}
	if h := resp.Header.Get("Docker-Content-Digest"); h != "" {
		n.Digest = h
	}

	if idx, err := v1.ParseIndexManifest(bytes.NewReader(body)); err == nil {
		n.Kind = "index"
		for _, child := range idx.Manifests {
			cn := node{
				Kind:         "descriptor",
				MediaType:    string(child.MediaType),
				Digest:       child.Digest.String(),
				Size:         child.Size,
				ArtifactType: child.ArtifactType,
				Annotations:  child.Annotations,
			}
			if child.Platform != nil {
				cn.Platform = child.Platform.String()
			}
			n.Children = append(n.Children, cn)
		}
		return n
	}

	if manifest, err := v1.ParseManifest(bytes.NewReader(body)); err == nil {
		n.Kind = "image"
		n.ConfigDigest = manifest.Config.Digest.String()
	}

	return n
}
