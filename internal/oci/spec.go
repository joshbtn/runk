package oci

import (
	"fmt"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"runk/internal/rootless"
)

type RunSpecInput struct {
	BundleDir   string
	RootFSPath  string
	ContainerID string
	Hostname    string
	Command     []string
	NetworkMode string
	IDMap       rootless.IDMap
}

func Build(input RunSpecInput) (*specs.Spec, error) {
	if len(input.Command) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	rootPath, err := filepath.Abs(input.RootFSPath)
	if err != nil {
		return nil, err
	}

	ns := []specs.LinuxNamespace{
		{Type: specs.UserNamespace},
		{Type: specs.PIDNamespace},
		{Type: specs.MountNamespace},
		{Type: specs.IPCNamespace},
		{Type: specs.UTSNamespace},
	}
	if input.NetworkMode != "host" {
		ns = append(ns, specs.LinuxNamespace{Type: specs.NetworkNamespace})
	}

	s := &specs.Spec{
		Version: specs.Version,
		Root: &specs.Root{
			Path:     rootPath,
			Readonly: false,
		},
		Process: &specs.Process{
			Args: input.Command,
			Env: []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"TERM=xterm",
				"HOME=/root",
				"USER=root",
				"LOGNAME=root",
			},
			Cwd:      "/",
			Terminal: true,
			User: specs.User{
				UID: 0,
				GID: 0,
			},
			NoNewPrivileges: true,
		},
		Hostname: input.Hostname,
		Linux: &specs.Linux{
			Namespaces: ns,
			UIDMappings: []specs.LinuxIDMapping{
				{ContainerID: 0, HostID: uint32(input.IDMap.UIDHostStart), Size: uint32(input.IDMap.Size)},
			},
			GIDMappings: []specs.LinuxIDMapping{
				{ContainerID: 0, HostID: uint32(input.IDMap.GIDHostStart), Size: uint32(input.IDMap.Size)},
			},
			MaskedPaths: []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
			},
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
		},
		Mounts: []specs.Mount{
			{Destination: "/proc", Type: "proc", Source: "proc"},
			{Destination: "/dev", Type: "tmpfs", Source: "tmpfs", Options: []string{"nosuid", "strictatime", "mode=755", "size=65536k"}},
			{Destination: "/dev/pts", Type: "devpts", Source: "devpts", Options: []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"}},
			{Destination: "/dev/shm", Type: "tmpfs", Source: "shm", Options: []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"}},
			{Destination: "/tmp", Type: "tmpfs", Source: "tmpfs", Options: []string{"nosuid", "nodev", "mode=1777", "size=65536k"}},
		},
		Annotations: map[string]string{
			"org.runk.poc":          "true",
			"org.runk.network_mode": input.NetworkMode,
		},
	}
	addBindFileMountIfExists(s, "/etc/resolv.conf")
	addBindFileMountIfExists(s, "/etc/hosts")
	addBindFileMountIfExists(s, "/etc/hostname")
	return s, nil
}

func addBindFileMountIfExists(spec *specs.Spec, path string) {
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return
	}
	spec.Mounts = append(spec.Mounts, specs.Mount{
		Destination: path,
		Type:        "bind",
		Source:      path,
		Options:     []string{"rbind", "ro"},
	})
}
