package rootless

import (
	"errors"
	"os/user"
	"strings"
	"testing"
)

func TestResolveIDMap_SubIDPreferred(t *testing.T) {
	origCurrentUser := currentUserFn
	origGeteuid := geteuidFn
	origGetegid := getegidFn
	origParseSubID := parseSubIDFileFn
	origHasHelpers := hasIDMapHelpersFn
	origLookPath := lookPathFn
	defer func() {
		currentUserFn = origCurrentUser
		geteuidFn = origGeteuid
		getegidFn = origGetegid
		parseSubIDFileFn = origParseSubID
		hasIDMapHelpersFn = origHasHelpers
		lookPathFn = origLookPath
	}()

	currentUserFn = func() (*user.User, error) { return &user.User{Username: "alice"}, nil }
	geteuidFn = func() int { return 1000 }
	getegidFn = func() int { return 1000 }
	parseSubIDFileFn = func(path, userName string) (int, int, bool) {
		if path == "/etc/subuid" {
			return 200000, 65536, true
		}
		return 300000, 65536, true
	}
	hasIDMapHelpersFn = func() bool { return true }
	lookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }

	idMap, warning, err := ResolveIDMap(false, false)
	if err != nil {
		t.Fatalf("ResolveIDMap returned unexpected error: %v", err)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
	if idMap.Strategy != StrategySubID {
		t.Fatalf("expected strategy %q, got %q", StrategySubID, idMap.Strategy)
	}
	if !idMap.UsingSubIDs {
		t.Fatalf("expected UsingSubIDs=true")
	}
	if idMap.Size != 65536 {
		t.Fatalf("expected size 65536, got %d", idMap.Size)
	}
}

func TestResolveIDMap_StrictDisablesFallback(t *testing.T) {
	origCurrentUser := currentUserFn
	origGeteuid := geteuidFn
	origGetegid := getegidFn
	origParseSubID := parseSubIDFileFn
	origHasHelpers := hasIDMapHelpersFn
	origLookPath := lookPathFn
	defer func() {
		currentUserFn = origCurrentUser
		geteuidFn = origGeteuid
		getegidFn = origGetegid
		parseSubIDFileFn = origParseSubID
		hasIDMapHelpersFn = origHasHelpers
		lookPathFn = origLookPath
	}()

	currentUserFn = func() (*user.User, error) { return &user.User{Username: "alice"}, nil }
	geteuidFn = func() int { return 1000 }
	getegidFn = func() int { return 1000 }
	parseSubIDFileFn = func(path, userName string) (int, int, bool) { return 0, 0, false }
	hasIDMapHelpersFn = func() bool { return false }
	lookPathFn = func(name string) (string, error) { return "", errors.New("not found") }

	_, _, err := ResolveIDMap(true, false)
	if err == nil {
		t.Fatalf("expected strict mode error")
	}
	if !strings.Contains(err.Error(), "strict mode enabled") {
		t.Fatalf("expected strict mode error text, got: %v", err)
	}
}

func TestResolveIDMap_ProotDefaultFallback(t *testing.T) {
	origCurrentUser := currentUserFn
	origGeteuid := geteuidFn
	origGetegid := getegidFn
	origParseSubID := parseSubIDFileFn
	origHasHelpers := hasIDMapHelpersFn
	origLookPath := lookPathFn
	defer func() {
		currentUserFn = origCurrentUser
		geteuidFn = origGeteuid
		getegidFn = origGetegid
		parseSubIDFileFn = origParseSubID
		hasIDMapHelpersFn = origHasHelpers
		lookPathFn = origLookPath
	}()

	currentUserFn = func() (*user.User, error) { return &user.User{Username: "alice"}, nil }
	geteuidFn = func() int { return 1000 }
	getegidFn = func() int { return 1000 }
	parseSubIDFileFn = func(path, userName string) (int, int, bool) { return 0, 0, false }
	hasIDMapHelpersFn = func() bool { return false }
	lookPathFn = func(name string) (string, error) {
		if name == "proot" {
			return "/usr/bin/proot", nil
		}
		return "", errors.New("not found")
	}

	idMap, warning, err := ResolveIDMap(false, false)
	if err != nil {
		t.Fatalf("ResolveIDMap returned unexpected error: %v", err)
	}
	if idMap.Strategy != StrategyProot {
		t.Fatalf("expected strategy %q, got %q", StrategyProot, idMap.Strategy)
	}
	if !strings.Contains(warning, "using proot fallback") {
		t.Fatalf("expected proot warning, got %q", warning)
	}
}

func TestResolveIDMap_ProotMissingFailsByDefault(t *testing.T) {
	origCurrentUser := currentUserFn
	origGeteuid := geteuidFn
	origGetegid := getegidFn
	origParseSubID := parseSubIDFileFn
	origHasHelpers := hasIDMapHelpersFn
	origLookPath := lookPathFn
	defer func() {
		currentUserFn = origCurrentUser
		geteuidFn = origGeteuid
		getegidFn = origGetegid
		parseSubIDFileFn = origParseSubID
		hasIDMapHelpersFn = origHasHelpers
		lookPathFn = origLookPath
	}()

	currentUserFn = func() (*user.User, error) { return &user.User{Username: "alice"}, nil }
	geteuidFn = func() int { return 1000 }
	getegidFn = func() int { return 1000 }
	parseSubIDFileFn = func(path, userName string) (int, int, bool) { return 0, 0, false }
	hasIDMapHelpersFn = func() bool { return false }
	lookPathFn = func(name string) (string, error) { return "", errors.New("not found") }

	_, _, err := ResolveIDMap(false, false)
	if err == nil {
		t.Fatalf("expected error when proot is missing")
	}
	if !strings.Contains(err.Error(), "install proot") {
		t.Fatalf("expected remediation text, got %v", err)
	}
}

func TestResolveIDMap_SingleUserFlagOverridesDefault(t *testing.T) {
	origCurrentUser := currentUserFn
	origGeteuid := geteuidFn
	origGetegid := getegidFn
	origParseSubID := parseSubIDFileFn
	origHasHelpers := hasIDMapHelpersFn
	origLookPath := lookPathFn
	defer func() {
		currentUserFn = origCurrentUser
		geteuidFn = origGeteuid
		getegidFn = origGetegid
		parseSubIDFileFn = origParseSubID
		hasIDMapHelpersFn = origHasHelpers
		lookPathFn = origLookPath
	}()

	currentUserFn = func() (*user.User, error) { return &user.User{Username: "alice"}, nil }
	geteuidFn = func() int { return 1001 }
	getegidFn = func() int { return 1002 }
	parseSubIDFileFn = func(path, userName string) (int, int, bool) { return 0, 0, false }
	hasIDMapHelpersFn = func() bool { return false }
	lookPathFn = func(name string) (string, error) { return "", errors.New("not found") }

	idMap, warning, err := ResolveIDMap(false, true)
	if err != nil {
		t.Fatalf("ResolveIDMap returned unexpected error: %v", err)
	}
	if idMap.Strategy != StrategySingleUser {
		t.Fatalf("expected strategy %q, got %q", StrategySingleUser, idMap.Strategy)
	}
	if idMap.UIDHostStart != 1001 || idMap.GIDHostStart != 1002 {
		t.Fatalf("unexpected mapped host ids: uid=%d gid=%d", idMap.UIDHostStart, idMap.GIDHostStart)
	}
	if !strings.Contains(warning, "single-user fallback") {
		t.Fatalf("expected single-user warning, got %q", warning)
	}
}
