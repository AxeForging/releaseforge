package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBumpMinorVersion(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "feature.go", "package main\n\nfunc Feature() {}\n")
	gitCommit(t, dir, "feat(core): add new feature")

	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("bump failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "v1.1.0") {
		t.Errorf("expected v1.1.0, got: %s", out)
	}
	if !strings.Contains(out, "MINOR") {
		t.Errorf("expected MINOR bump, got: %s", out)
	}
}

func TestBumpPatchVersion(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "fix.go", "package main\n\nfunc Fix() {}\n")
	gitCommit(t, dir, "fix: resolve null pointer")

	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("bump failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "v1.0.1") {
		t.Errorf("expected v1.0.1, got: %s", out)
	}
	if !strings.Contains(out, "PATCH") {
		t.Errorf("expected PATCH bump, got: %s", out)
	}
}

func TestBumpMajorVersion(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "breaking.go", "package main\n\nfunc Breaking() {}\n")
	gitCommit(t, dir, "feat!: redesign API")

	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("bump failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "v2.0.0") {
		t.Errorf("expected v2.0.0, got: %s", out)
	}
	if !strings.Contains(out, "MAJOR") {
		t.Errorf("expected MAJOR bump, got: %s", out)
	}
}

func TestBumpQuietMode(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "new.go", "package main\n")
	gitCommit(t, dir, "feat: add feature")

	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0", "--quiet")
	if err != nil {
		t.Fatalf("bump quiet failed: %v\n%s", err, out)
	}

	// In quiet mode, output should contain the version but not the full report
	if !strings.Contains(out, "v1.1.0") {
		t.Errorf("expected v1.1.0, got: %s", out)
	}
	if strings.Contains(out, "Conventional Commit Analysis") {
		t.Errorf("quiet mode should not print report header, got: %s", out)
	}
}

func TestBumpAutoDetectTag(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "new.go", "package main\n")
	gitCommit(t, dir, "fix: bug fix")

	out, err := runBinaryInDir(t, bin, dir, "bump")
	if err != nil {
		t.Fatalf("bump auto-detect failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Auto-detected base tag: v1.0.0") {
		t.Errorf("expected auto-detect message, got: %s", out)
	}
	if !strings.Contains(out, "v1.0.1") {
		t.Errorf("expected v1.0.1, got: %s", out)
	}
}

func TestBumpOutputJSON(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "new.go", "package main\n")
	gitCommit(t, dir, "feat(api): add endpoint")

	jsonPath := filepath.Join(dir, "analysis.json")
	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0", "--output-json", jsonPath)
	if err != nil {
		t.Fatalf("bump json output failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["next_version"] != "v1.1.0" {
		t.Errorf("JSON next_version = %v, want v1.1.0", result["next_version"])
	}
	if result["bump_level"] != "minor" {
		t.Errorf("JSON bump_level = %v, want minor", result["bump_level"])
	}
	if result["base_version"] != "v1.0.0" {
		t.Errorf("JSON base_version = %v, want v1.0.0", result["base_version"])
	}
}

func TestBumpOutputVersion(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "new.go", "package main\n")
	gitCommit(t, dir, "fix: patch fix")

	versionPath := filepath.Join(dir, "next-version.txt")
	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0", "--output-version", versionPath)
	if err != nil {
		t.Fatalf("bump version output failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}

	if string(data) != "v1.0.1" {
		t.Errorf("version file = %q, want v1.0.1", string(data))
	}
}

func TestBumpMultipleCommitTypes(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")
	gitTag(t, dir, "v2.1.0")

	// Add a mix of commit types
	writeFile(t, dir, "a.go", "package main\n")
	gitCommit(t, dir, "feat(ui): add dashboard")

	writeFile(t, dir, "b.go", "package main\n")
	gitCommit(t, dir, "fix(api): handle timeout")

	writeFile(t, dir, "c.go", "package main\n")
	gitCommit(t, dir, "docs: update README")

	writeFile(t, dir, "d.go", "package main\n")
	gitCommit(t, dir, "chore: update deps")

	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v2.1.0")
	if err != nil {
		t.Fatalf("bump multi failed: %v\n%s", err, out)
	}

	// feat is the highest -> minor bump
	if !strings.Contains(out, "v2.2.0") {
		t.Errorf("expected v2.2.0 (minor from feat), got: %s", out)
	}

	// Report should categorize commits
	if !strings.Contains(out, "Features") {
		t.Errorf("expected Features section, got: %s", out)
	}
	if !strings.Contains(out, "Bug Fixes") {
		t.Errorf("expected Bug Fixes section, got: %s", out)
	}
}

func TestBumpNoCommits(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")
	gitTag(t, dir, "v1.0.0")

	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("bump no commits failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "No commits found") {
		t.Errorf("expected no commits message, got: %s", out)
	}
}

func TestBumpInvalidTag(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")

	_, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "not-semver")
	if err == nil {
		t.Error("expected error for invalid semver tag")
	}
}

func TestBumpWithVPrefix(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "new.go", "package main\n")
	gitCommit(t, dir, "feat: add feature")

	out, err := runBinaryInDir(t, bin, dir, "bump", "--tag", "v1.0.0", "--quiet")
	if err != nil {
		t.Fatalf("bump failed: %v\n%s", err, out)
	}

	// Should preserve v prefix
	if !strings.Contains(out, "v1.1.0") {
		t.Errorf("expected v1.1.0 (with v prefix), got: %s", out)
	}
}
