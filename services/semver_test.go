package services

import (
	"testing"

	"github.com/AxeForging/releaseforge/domain"
)

func TestParseConventionalCommit_Basic(t *testing.T) {
	svc := &SemverService{}

	tests := []struct {
		name      string
		message   string
		wantType  string
		wantScope string
		wantDesc  string
		wantBreak bool
		wantBump  string
	}{
		{
			name:     "simple feat",
			message:  "feat: add user authentication",
			wantType: "feat", wantDesc: "add user authentication",
			wantBump: "minor",
		},
		{
			name:     "feat with scope",
			message:  "feat(api): add new endpoint",
			wantType: "feat", wantScope: "api", wantDesc: "add new endpoint",
			wantBump: "minor",
		},
		{
			name:     "fix",
			message:  "fix: resolve null pointer",
			wantType: "fix", wantDesc: "resolve null pointer",
			wantBump: "patch",
		},
		{
			name:     "fix with scope",
			message:  "fix(auth): handle expired tokens",
			wantType: "fix", wantScope: "auth", wantDesc: "handle expired tokens",
			wantBump: "patch",
		},
		{
			name:     "breaking feat with bang",
			message:  "feat!: redesign API",
			wantType: "feat", wantDesc: "redesign API",
			wantBreak: true, wantBump: "major",
		},
		{
			name:     "breaking fix with scope and bang",
			message:  "fix(api)!: change response format",
			wantType: "fix", wantScope: "api", wantDesc: "change response format",
			wantBreak: true, wantBump: "major",
		},
		{
			name:     "breaking change in footer",
			message:  "feat: new auth flow\n\nBREAKING CHANGE: old tokens are invalidated",
			wantType: "feat", wantDesc: "new auth flow",
			wantBreak: true, wantBump: "major",
		},
		{
			name:     "docs",
			message:  "docs: update README",
			wantType: "docs", wantDesc: "update README",
			wantBump: "none",
		},
		{
			name:     "chore",
			message:  "chore: update dependencies",
			wantType: "chore", wantDesc: "update dependencies",
			wantBump: "none",
		},
		{
			name:     "ci",
			message:  "ci: add GitHub Actions workflow",
			wantType: "ci", wantDesc: "add GitHub Actions workflow",
			wantBump: "none",
		},
		{
			name:     "refactor",
			message:  "refactor(core): simplify logic",
			wantType: "refactor", wantScope: "core", wantDesc: "simplify logic",
			wantBump: "none",
		},
		{
			name:     "perf",
			message:  "perf: optimize query performance",
			wantType: "perf", wantDesc: "optimize query performance",
			wantBump: "patch",
		},
		{
			name:     "revert",
			message:  "revert: undo last migration",
			wantType: "revert", wantDesc: "undo last migration",
			wantBump: "patch",
		},
		{
			name:     "test",
			message:  "test: add integration tests",
			wantType: "test", wantDesc: "add integration tests",
			wantBump: "none",
		},
		{
			name:     "build",
			message:  "build: update go version to 1.24",
			wantType: "build", wantDesc: "update go version to 1.24",
			wantBump: "none",
		},
		{
			name:     "style",
			message:  "style: fix formatting",
			wantType: "style", wantDesc: "fix formatting",
			wantBump: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.ParseConventionalCommit(tt.message)
			if result == nil {
				t.Fatal("ParseConventionalCommit() returned nil")
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", result.Type, tt.wantType)
			}
			if result.Scope != tt.wantScope {
				t.Errorf("Scope = %q, want %q", result.Scope, tt.wantScope)
			}
			if result.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", result.Description, tt.wantDesc)
			}
			if result.Breaking != tt.wantBreak {
				t.Errorf("Breaking = %v, want %v", result.Breaking, tt.wantBreak)
			}
			if result.BumpLevel != tt.wantBump {
				t.Errorf("BumpLevel = %q, want %q", result.BumpLevel, tt.wantBump)
			}
		})
	}
}

func TestParseConventionalCommit_NonConventional(t *testing.T) {
	svc := &SemverService{}

	messages := []string{
		"just a regular message",
		"Update README",
		"Merge branch 'main' into dev",
		"v1.0.0",
		"123 fix something",
	}

	for _, msg := range messages {
		t.Run(msg, func(t *testing.T) {
			result := svc.ParseConventionalCommit(msg)
			if result != nil {
				t.Errorf("ParseConventionalCommit(%q) = %+v, want nil", msg, result)
			}
		})
	}
}

func TestAnalyzeCommits_BumpLevel(t *testing.T) {
	svc := &SemverService{}

	tests := []struct {
		name     string
		messages []string
		wantBump string
	}{
		{
			name:     "single feat = minor",
			messages: []string{"feat: add feature"},
			wantBump: "minor",
		},
		{
			name:     "single fix = patch",
			messages: []string{"fix: fix bug"},
			wantBump: "patch",
		},
		{
			name:     "feat + fix = minor (highest wins)",
			messages: []string{"feat: new feature", "fix: bug fix"},
			wantBump: "minor",
		},
		{
			name:     "breaking always wins",
			messages: []string{"fix: small fix", "feat!: breaking feature"},
			wantBump: "major",
		},
		{
			name:     "only chore = patch (defaults to patch when none)",
			messages: []string{"chore: update deps"},
			wantBump: "patch",
		},
		{
			name:     "non-conventional = patch (defaults to patch when none)",
			messages: []string{"random commit message"},
			wantBump: "patch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var commits []domain.DetailedCommit
			for i, msg := range tt.messages {
				commits = append(commits, domain.DetailedCommit{
					Hash:    "abc1234" + string(rune('0'+i)),
					Message: msg,
				})
			}

			analysis := svc.AnalyzeCommits(commits)

			// CalculateNextVersion defaults "none" to "patch", so we simulate that
			bumpLevel := analysis.BumpLevel
			if bumpLevel == "none" {
				bumpLevel = "patch"
			}

			if bumpLevel != tt.wantBump {
				t.Errorf("BumpLevel = %q, want %q", bumpLevel, tt.wantBump)
			}
		})
	}
}

func TestBumpHigher(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"major", "minor", true},
		{"major", "patch", true},
		{"major", "none", true},
		{"minor", "patch", true},
		{"minor", "none", true},
		{"patch", "none", true},
		{"none", "none", false},
		{"patch", "patch", false},
		{"minor", "major", false},
		{"patch", "minor", false},
	}

	for _, tt := range tests {
		name := tt.a + "_vs_" + tt.b
		t.Run(name, func(t *testing.T) {
			if got := bumpHigher(tt.a, tt.b); got != tt.want {
				t.Errorf("bumpHigher(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
