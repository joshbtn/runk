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
	Strategy     string
}

const (
	StrategySubID      = "subid"
	StrategySingleUser = "single-user"
	StrategyProot      = "proot"
)

var (
	currentUserFn     = user.Current
	geteuidFn         = os.Geteuid
	getegidFn         = os.Getegid
	parseSubIDFileFn  = parseSubIDFile
	hasIDMapHelpersFn = hasIDMapHelpers
	lookPathFn        = exec.LookPath
)

func ResolveIDMap(strict bool, singleUserFallback bool) (IDMap, string, error) {
	u, err := currentUserFn()
	if err != nil {
		return IDMap{}, "", fmt.Errorf("resolve current user: %w", err)
	}

	euid := geteuidFn()
	egid := getegidFn()

	uidStart, uidCount, uidOK := parseSubIDFileFn("/etc/subuid", u.Username)
	gidStart, gidCount, gidOK := parseSubIDFileFn("/etc/subgid", u.Username)

	if uidOK && gidOK {
		if hasIDMapHelpersFn() {
			size := min(uidCount, gidCount)
			if size < 1 {
				return IDMap{}, "", fmt.Errorf("invalid subid range for user %q", u.Username)
			}
			if size > 65536 {
				size = 65536
			}
			return IDMap{UIDHostStart: uidStart, GIDHostStart: gidStart, Size: size, UsingSubIDs: true, Strategy: StrategySubID}, "", nil
		}

		if strict {
			return IDMap{}, "", fmt.Errorf("subuid/subgid present for user %q but newuidmap/newgidmap are missing and strict mode is enabled", u.Username)
		}
		return resolveFallback(euid, egid, singleUserFallback, "subuid/subgid found but newuidmap/newgidmap are missing")
	}

	if strict {
		return IDMap{}, "", fmt.Errorf("missing subuid/subgid for user %q and strict mode enabled", u.Username)
	}

	return resolveFallback(euid, egid, singleUserFallback, "subuid/subgid not found")
}

func resolveFallback(euid, egid int, singleUserFallback bool, reason string) (IDMap, string, error) {
	if singleUserFallback {
		warning := reason + "; using single-user fallback mapping (container root -> current user)"
		return IDMap{UIDHostStart: euid, GIDHostStart: egid, Size: 1, UsingSubIDs: false, Strategy: StrategySingleUser}, warning, nil
	}

	if _, err := lookPathFn("proot"); err != nil {
		return IDMap{}, "", fmt.Errorf("%s; default proot fallback requires 'proot' binary (install proot or retry with --single-user-fallback)", reason)
	}

	warning := reason + "; using proot fallback by default"
	return IDMap{UIDHostStart: euid, GIDHostStart: egid, Size: 1, UsingSubIDs: false, Strategy: StrategyProot}, warning, nil
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
	if _, err := lookPathFn("newuidmap"); err != nil {
		return false
	}
	if _, err := lookPathFn("newgidmap"); err != nil {
		return false
	}
	return true
}
