package integration

import (
	"strings"
	"testing"
)

func TestCheckAllConventional(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "a.go", "package main\n")
	gitCommit(t, dir, "feat: add feature")

	writeFile(t, dir, "b.go", "package main\n")
	gitCommit(t, dir, "fix: bug fix")

	out, err := runBinaryInDir(t, bin, dir, "check", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("check failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "All commits follow conventional commit format") {
		t.Errorf("expected all conventional, got: %s", out)
	}
	if !strings.Contains(out, "Non-conventional:    0") {
		t.Errorf("expected 0 non-conventional, got: %s", out)
	}
}

func TestCheckNonConventionalCommits(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "a.go", "package main\n")
	gitCommit(t, dir, "feat: proper commit")

	writeFile(t, dir, "b.go", "package main\n")
	gitCommit(t, dir, "just a random message")

	writeFile(t, dir, "c.go", "package main\n")
	gitCommit(t, dir, "Update something")

	out, err := runBinaryInDir(t, bin, dir, "check", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("check failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Non-conventional:    2") {
		t.Errorf("expected 2 non-conventional, got: %s", out)
	}
	if !strings.Contains(out, "Non-conventional commits:") {
		t.Errorf("expected non-conventional list, got: %s", out)
	}
}

func TestCheckStrictModeFails(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "a.go", "package main\n")
	gitCommit(t, dir, "random non-conventional commit")

	_, err := runBinaryInDir(t, bin, dir, "check", "--tag", "v1.0.0", "--strict")
	if err == nil {
		t.Error("expected strict mode to fail with non-conventional commits")
	}
}

func TestCheckStrictModePassesWhenClean(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "a.go", "package main\n")
	gitCommit(t, dir, "feat: proper feature")

	writeFile(t, dir, "b.go", "package main\n")
	gitCommit(t, dir, "fix(auth): handle expired token")

	out, err := runBinaryInDir(t, bin, dir, "check", "--tag", "v1.0.0", "--strict")
	if err != nil {
		t.Fatalf("strict mode should pass with conventional commits: %v\n%s", err, out)
	}

	if !strings.Contains(out, "All commits follow conventional commit format") {
		t.Errorf("expected all conventional, got: %s", out)
	}
}
