package remote

import (
	"strings"
	"testing"

	"github.com/peacock0803sz/notizen/internal/config"
)

func baseCfg() *config.RemoteConfig {
	return &config.RemoteConfig{
		Host: "example.com",
		Port: 22,
	}
}

func TestBuildArgs_RecursiveUsesArchive(t *testing.T) {
	args := BuildArgs(baseCfg(), "/src", false, false, true)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--archive") {
		t.Errorf("recursive mode should include --archive, got: %s", joined)
	}
	if strings.Contains(joined, "-lptoD") {
		t.Errorf("recursive mode should not include -lptoD, got: %s", joined)
	}
}

func TestBuildArgs_NonRecursiveExplicitFlags(t *testing.T) {
	args := BuildArgs(baseCfg(), "/src", false, false, false)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-lptoD") {
		t.Errorf("non-recursive mode should include -lptoD, got: %s", joined)
	}
	if strings.Contains(joined, "--archive") {
		t.Errorf("non-recursive mode should not include --archive, got: %s", joined)
	}
}

func TestBuildArgs_VerboseCompressAlwaysPresent(t *testing.T) {
	for _, recursive := range []bool{true, false} {
		args := BuildArgs(baseCfg(), "/src", false, false, recursive)
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--verbose") {
			t.Errorf("--verbose missing (recursive=%v): %s", recursive, joined)
		}
		if !strings.Contains(joined, "--compress") {
			t.Errorf("--compress missing (recursive=%v): %s", recursive, joined)
		}
	}
}

func TestBuildArgs_DeleteFlag(t *testing.T) {
	with := BuildArgs(baseCfg(), "/src", false, true, true)
	without := BuildArgs(baseCfg(), "/src", false, false, true)
	if !strings.Contains(strings.Join(with, " "), "--delete") {
		t.Error("expected --delete when delete=true")
	}
	if strings.Contains(strings.Join(without, " "), "--delete") {
		t.Error("unexpected --delete when delete=false")
	}
}

func TestBuildArgs_DryRunFlag(t *testing.T) {
	args := BuildArgs(baseCfg(), "/src", true, false, true)
	if !strings.Contains(strings.Join(args, " "), "--dry-run") {
		t.Error("expected --dry-run flag")
	}
}

func TestBuildArgs_CustomPortAndKey(t *testing.T) {
	cfg := &config.RemoteConfig{Host: "h", Port: 2222, Key: "/path/to/key"}
	args := BuildArgs(cfg, "/src", false, false, true)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-p 2222") {
		t.Errorf("expected -p 2222 in SSH command, got: %s", joined)
	}
	if !strings.Contains(joined, `-i "/path/to/key"`) {
		t.Errorf(`expected -i "/path/to/key" in SSH command, got: %s`, joined)
	}
}

func TestBuildArgs_DoubleDashSeparator(t *testing.T) {
	args := BuildArgs(baseCfg(), "/src", false, false, true)
	// "--" must appear before src and dest
	var ddIdx int
	for i, a := range args {
		if a == "--" {
			ddIdx = i
			break
		}
	}
	if ddIdx == 0 {
		t.Fatalf("expected -- separator in args: %v", args)
	}
	if args[ddIdx+1] != "/src" {
		t.Errorf("src should follow --, got %q", args[ddIdx+1])
	}
}

func TestBuildArgs_UserAtHost(t *testing.T) {
	cfg := &config.RemoteConfig{Host: "example.com", User: "deploy", Port: 22, Path: "/var/www"}
	args := BuildArgs(cfg, "/src", false, false, true)
	dest := args[len(args)-1]
	if dest != "deploy@example.com:/var/www" {
		t.Errorf("dest = %q, want deploy@example.com:/var/www", dest)
	}
}
