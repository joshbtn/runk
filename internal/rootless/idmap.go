package rootless

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

type IDMap struct {
	UIDHostStart int
	GIDHostStart int
	Size         int
	UsingSubIDs  bool
}

func ResolveIDMap(strict bool) (IDMap, string, error) {
	u, err := user.Current()
	if err != nil {
		return IDMap{}, "", fmt.Errorf("resolve current user: %w", err)
	}

	euid := os.Geteuid()
	egid := os.Getegid()

	uidStart, uidCount, uidOK := parseSubIDFile("/etc/subuid", u.Username)
	gidStart, gidCount, gidOK := parseSubIDFile("/etc/subgid", u.Username)

	if uidOK && gidOK {
		if hasIDMapHelpers() {
			size := min(uidCount, gidCount)
			if size < 1 {
				return IDMap{}, "", fmt.Errorf("invalid subid range for user %q", u.Username)
			}
			if size > 65536 {
				size = 65536
			}
			return IDMap{UIDHostStart: uidStart, GIDHostStart: gidStart, Size: size, UsingSubIDs: true}, "", nil
		}

		if strict {
			return IDMap{}, "", fmt.Errorf("subuid/subgid present for user %q but newuidmap/newgidmap are missing", u.Username)
		}

		warning := "subuid/subgid found but newuidmap/newgidmap are missing; using single-UID/GID fallback mapping (container root -> current user)"
		return IDMap{UIDHostStart: euid, GIDHostStart: egid, Size: 1, UsingSubIDs: false}, warning, nil
	}

	if strict {
		return IDMap{}, "", fmt.Errorf("missing subuid/subgid for user %q and strict mode enabled", u.Username)
	}

	warning := "subuid/subgid not found; using single-UID/GID fallback mapping (container root -> current user)"
	return IDMap{UIDHostStart: euid, GIDHostStart: egid, Size: 1, UsingSubIDs: false}, warning, nil
}

func parseSubIDFile(path, userName string) (start int, count int, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 3 || parts[0] != userName {
			continue
		}
		start, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, false
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, false
		}
		return start, count, true
	}
	return 0, 0, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func hasIDMapHelpers() bool {
	if _, err := exec.LookPath("newuidmap"); err != nil {
		return false
	}
	if _, err := exec.LookPath("newgidmap"); err != nil {
		return false
	}
	return true
}
