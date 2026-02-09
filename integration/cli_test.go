package integration

import (
	"strings"
	"testing"
)

func TestCLIVersionCommand(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "version")
	if err != nil {
		t.Fatalf("version command failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "releaseforge version") {
		t.Errorf("expected version output, got: %s", out)
	}
	if !strings.Contains(out, "Build time:") {
		t.Errorf("expected build time, got: %s", out)
	}
	if !strings.Contains(out, "Git commit:") {
		t.Errorf("expected git commit, got: %s", out)
	}
}

func TestCLIHelpOutput(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "--help")
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}

	expected := []string{"generate", "bump", "check", "templates", "version"}
	for _, cmd := range expected {
		if !strings.Contains(out, cmd) {
			t.Errorf("help output missing command %q: %s", cmd, out)
		}
	}
}

func TestCLIDefaultAction(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin)
	if err != nil {
		t.Fatalf("default action failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Commands:") {
		t.Errorf("expected commands list, got: %s", out)
	}
	if !strings.Contains(out, "generate") {
		t.Errorf("expected generate in output, got: %s", out)
	}
	if !strings.Contains(out, "bump") {
		t.Errorf("expected bump in output, got: %s", out)
	}
}

func TestCLIGenerateHelp(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "generate", "--help")
	if err != nil {
		t.Fatalf("generate help failed: %v\n%s", err, out)
	}

	flags := []string{"--provider", "--model", "--key", "--git-tag", "--force-git-mode", "--template-name", "--output"}
	for _, flag := range flags {
		if !strings.Contains(out, flag) {
			t.Errorf("generate help missing flag %q", flag)
		}
	}
}

func TestCLIBumpHelp(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "bump", "--help")
	if err != nil {
		t.Fatalf("bump help failed: %v\n%s", err, out)
	}

	flags := []string{"--tag", "--branch", "--max-commits", "--output-json", "--output-version", "--quiet"}
	for _, flag := range flags {
		if !strings.Contains(out, flag) {
			t.Errorf("bump help missing flag %q", flag)
		}
	}

	// Check that the description documents conventional commit types
	types := []string{"feat", "fix", "perf", "revert", "docs", "style", "refactor", "test", "build", "ci", "chore"}
	for _, typ := range types {
		if !strings.Contains(out, typ) {
			t.Errorf("bump help missing commit type documentation for %q", typ)
		}
	}
}

func TestCLICheckHelp(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "check", "--help")
	if err != nil {
		t.Fatalf("check help failed: %v\n%s", err, out)
	}

	flags := []string{"--tag", "--branch", "--strict"}
	for _, flag := range flags {
		if !strings.Contains(out, flag) {
			t.Errorf("check help missing flag %q", flag)
		}
	}
}

func TestCLITemplatesCommand(t *testing.T) {
	bin := buildBinary(t)

	out, err := runBinary(t, bin, "templates")
	if err != nil {
		t.Fatalf("templates command failed: %v\n%s", err, out)
	}

	templates := []string{"semver-release-notes", "conventional-changelog", "version-analysis"}
	for _, tpl := range templates {
		if !strings.Contains(out, tpl) {
			t.Errorf("templates output missing %q", tpl)
		}
	}
}
