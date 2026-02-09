package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateForceGitMode(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)

	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release of the CLI tool")

	writeFile(t, dir, "feature.go", "package main\n\nfunc Feature() {}\n")
	gitCommit(t, dir, "feat(core): add new feature")

	writeFile(t, dir, "fix.go", "package main\n\nfunc Fix() {}\n")
	gitCommit(t, dir, "fix: resolve null pointer issue")

	outputPath := filepath.Join(dir, "release-notes.md")
	out, err := runBinaryInDir(t, bin, dir, "generate", "--force-git-mode", "-o", outputPath)
	if err != nil {
		t.Fatalf("generate force-git-mode failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("output file is empty")
	}

	// Should contain categorized commits
	if !strings.Contains(content, "Features") && !strings.Contains(content, "feat") {
		t.Errorf("expected features section in output: %s", content)
	}
}

func TestGenerateForceGitModeWithTag(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)

	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial release")
	gitTag(t, dir, "v1.0.0")

	writeFile(t, dir, "new.go", "package main\n")
	gitCommit(t, dir, "feat: add new feature after tag")

	outputPath := filepath.Join(dir, "notes.md")
	out, err := runBinaryInDir(t, bin, dir, "generate", "--force-git-mode", "--git-tag", "v1.0.0", "--analyze-from-tag", "-o", outputPath)
	if err != nil {
		t.Fatalf("generate with tag failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateForceGitModeCreatesJSON(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial feature")

	outputPath := filepath.Join(dir, "notes.md")
	out, err := runBinaryInDir(t, bin, dir, "generate", "--force-git-mode", "-o", outputPath)
	if err != nil {
		t.Fatalf("generate failed: %v\n%s", err, out)
	}

	// Should create the markdown output
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to exist")
	}
}

func TestGenerateWithTemplateName(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial feature")

	outputPath := filepath.Join(dir, "notes.md")
	out, err := runBinaryInDir(t, bin, dir, "generate", "--force-git-mode", "--template-name", "conventional-changelog", "-o", outputPath)
	if err != nil {
		t.Fatalf("generate with template failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateNoAPIKeyFails(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main\n")
	gitCommit(t, dir, "feat: initial")

	// Without force-git-mode and without API key, should fail
	_, err := runBinaryInDir(t, bin, dir, "generate")
	if err == nil {
		t.Error("expected error when no API key provided and not in force-git-mode")
	}
}
