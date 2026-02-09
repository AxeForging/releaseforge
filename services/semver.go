package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AxeForging/releaseforge/domain"
	"github.com/AxeForging/releaseforge/helpers"
)

// ConventionalCommitType represents the type of a conventional commit
type ConventionalCommitType struct {
	Type        string
	Scope       string
	Description string
	Breaking    bool
	BumpLevel   string // "major", "minor", "patch", "none"
}

// SemverService handles semantic versioning analysis based on conventional commits
type SemverService struct {
	git *GitService
}

func NewSemverService(git *GitService) *SemverService {
	return &SemverService{git: git}
}

// ConventionalCommitPatterns defines how commit types map to semver bumps
//
// Conventional Commit Format: <type>[optional scope][!]: <description>
//
// Major (breaking changes):
//   - Any commit with "!" after the type/scope (e.g., "feat!:", "fix(api)!:")
//   - Any commit with "BREAKING CHANGE:" in the footer
//
// Minor (new features):
//   - feat: A new feature
//
// Patch (bug fixes and small changes):
//   - fix: A bug fix
//   - perf: A performance improvement
//   - revert: Reverts a previous commit
//
// No version bump (non-functional changes):
//   - docs: Documentation only changes
//   - style: Code style changes (formatting, whitespace)
//   - refactor: Code refactoring (no bug fix or feature)
//   - test: Adding or correcting tests
//   - build: Build system or external dependency changes
//   - ci: CI/CD configuration changes
//   - chore: Maintenance tasks
var conventionalTypes = map[string]string{
	"feat":     "minor",
	"fix":      "patch",
	"perf":     "patch",
	"revert":   "patch",
	"docs":     "none",
	"style":    "none",
	"refactor": "none",
	"test":     "none",
	"build":    "none",
	"ci":       "none",
	"chore":    "none",
}

// ParseConventionalCommit parses a commit message following the Conventional Commits spec
// Format: <type>[optional scope][!]: <description>
func (s *SemverService) ParseConventionalCommit(message string) *ConventionalCommitType {
	firstLine := message
	if idx := strings.Index(message, "\n"); idx >= 0 {
		firstLine = message[:idx]
	}
	firstLine = strings.TrimSpace(firstLine)

	// Pattern: type(scope)!: description  OR  type!: description  OR  type(scope): description  OR  type: description
	re := regexp.MustCompile(`^([a-zA-Z]+)(?:\(([^)]*)\))?(!)?\s*:\s*(.*)$`)
	match := re.FindStringSubmatch(firstLine)
	if match == nil {
		return nil
	}

	commitType := strings.ToLower(match[1])
	scope := match[2]
	breaking := match[3] == "!"
	description := match[4]

	// Check for BREAKING CHANGE in footer
	if !breaking && strings.Contains(strings.ToUpper(message), "BREAKING CHANGE:") {
		breaking = true
	}

	bumpLevel, known := conventionalTypes[commitType]
	if !known {
		bumpLevel = "none"
	}

	// Breaking changes always bump major
	if breaking {
		bumpLevel = "major"
	}

	return &ConventionalCommitType{
		Type:        commitType,
		Scope:       scope,
		Description: description,
		Breaking:    breaking,
		BumpLevel:   bumpLevel,
	}
}

// AnalyzeCommits analyzes a list of commits and determines the appropriate semver bump
func (s *SemverService) AnalyzeCommits(commits []domain.DetailedCommit) *domain.BumpAnalysis {
	analysis := &domain.BumpAnalysis{
		BumpLevel:  "none",
		Commits:    make(map[string][]domain.AnalyzedCommit),
		TotalCount: len(commits),
	}

	for _, c := range commits {
		parsed := s.ParseConventionalCommit(c.Message)

		var ac domain.AnalyzedCommit
		if parsed != nil {
			ac = domain.AnalyzedCommit{
				Hash:        shortHash(c.Hash),
				Message:     firstLine(c.Message),
				Type:        parsed.Type,
				Scope:       parsed.Scope,
				Description: parsed.Description,
				Breaking:    parsed.Breaking,
				BumpLevel:   parsed.BumpLevel,
				Conventional: true,
			}
		} else {
			ac = domain.AnalyzedCommit{
				Hash:         shortHash(c.Hash),
				Message:      firstLine(c.Message),
				Type:         "unknown",
				BumpLevel:    "none",
				Conventional: false,
			}
			analysis.NonConventionalCount++
		}

		analysis.Commits[ac.Type] = append(analysis.Commits[ac.Type], ac)

		// Update overall bump level (highest wins)
		if bumpHigher(ac.BumpLevel, analysis.BumpLevel) {
			analysis.BumpLevel = ac.BumpLevel
		}
	}

	return analysis
}

// CalculateNextVersion computes the next version given a base tag and the bump analysis
func (s *SemverService) CalculateNextVersion(baseTag string, analysis *domain.BumpAnalysis) (string, error) {
	parsed := s.git.ParseSemverTag(baseTag)
	if parsed == nil {
		return "", fmt.Errorf("cannot parse %q as semver tag", baseTag)
	}

	if analysis.BumpLevel == "none" {
		helpers.Log.Warn().Msg("No version-bumping commits found. Defaulting to patch bump.")
		analysis.BumpLevel = "patch"
	}

	next := *parsed
	switch analysis.BumpLevel {
	case "major":
		next.Major++
		next.Minor = 0
		next.Patch = 0
	case "minor":
		next.Minor++
		next.Patch = 0
	case "patch":
		next.Patch++
	}

	prefix := ""
	if strings.HasPrefix(baseTag, "v") {
		prefix = "v"
	}
	return fmt.Sprintf("%s%d.%d.%d", prefix, next.Major, next.Minor, next.Patch), nil
}

// FormatAnalysisReport produces a human-readable report of the commit analysis
func (s *SemverService) FormatAnalysisReport(baseTag, nextVersion string, analysis *domain.BumpAnalysis) string {
	var sb strings.Builder

	sb.WriteString("Conventional Commit Analysis\n")
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(fmt.Sprintf("  Base version:     %s\n", baseTag))
	sb.WriteString(fmt.Sprintf("  Suggested bump:   %s\n", strings.ToUpper(analysis.BumpLevel)))
	sb.WriteString(fmt.Sprintf("  Next version:     %s\n", nextVersion))
	sb.WriteString(fmt.Sprintf("  Total commits:    %d\n", analysis.TotalCount))
	if analysis.NonConventionalCount > 0 {
		sb.WriteString(fmt.Sprintf("  Non-conventional: %d\n", analysis.NonConventionalCount))
	}
	sb.WriteString("\n")

	// Order: breaking first, then by impact
	typeOrder := []string{"feat", "fix", "perf", "revert", "refactor", "docs", "style", "test", "build", "ci", "chore", "unknown"}
	typeLabels := map[string]string{
		"feat":     "Features (minor)",
		"fix":      "Bug Fixes (patch)",
		"perf":     "Performance (patch)",
		"revert":   "Reverts (patch)",
		"refactor": "Refactoring (no bump)",
		"docs":     "Documentation (no bump)",
		"style":    "Style (no bump)",
		"test":     "Tests (no bump)",
		"build":    "Build (no bump)",
		"ci":       "CI/CD (no bump)",
		"chore":    "Chores (no bump)",
		"unknown":  "Non-conventional",
	}

	// Print breaking changes first
	var breakingCommits []domain.AnalyzedCommit
	for _, commits := range analysis.Commits {
		for _, c := range commits {
			if c.Breaking {
				breakingCommits = append(breakingCommits, c)
			}
		}
	}
	if len(breakingCommits) > 0 {
		sb.WriteString("  BREAKING CHANGES (major):\n")
		for _, c := range breakingCommits {
			scope := ""
			if c.Scope != "" {
				scope = fmt.Sprintf("(%s)", c.Scope)
			}
			sb.WriteString(fmt.Sprintf("    - %s %s%s: %s\n", c.Hash, c.Type, scope, c.Description))
		}
		sb.WriteString("\n")
	}

	// Print by type
	for _, t := range typeOrder {
		commits, ok := analysis.Commits[t]
		if !ok || len(commits) == 0 {
			continue
		}

		// Skip breaking commits already printed
		var nonBreaking []domain.AnalyzedCommit
		for _, c := range commits {
			if !c.Breaking {
				nonBreaking = append(nonBreaking, c)
			}
		}
		if len(nonBreaking) == 0 {
			continue
		}

		label := typeLabels[t]
		if label == "" {
			label = t
		}
		sb.WriteString(fmt.Sprintf("  %s:\n", label))
		for _, c := range nonBreaking {
			scope := ""
			if c.Scope != "" {
				scope = fmt.Sprintf("(%s)", c.Scope)
			}
			desc := c.Description
			if desc == "" {
				desc = c.Message
			}
			sb.WriteString(fmt.Sprintf("    - %s %s: %s\n", c.Hash, scope, desc))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("-", 50) + "\n")
	sb.WriteString(fmt.Sprintf("Recommendation: %s -> %s (%s bump)\n", baseTag, nextVersion, analysis.BumpLevel))

	return sb.String()
}

// GetCommitsBetween gets commits between a tag and a branch/ref
func (s *SemverService) GetCommitsBetween(fromTag, toRef string, maxCommits int) ([]domain.DetailedCommit, error) {
	rangeArg := fmt.Sprintf("%s..%s", fromTag, toRef)
	helpers.Log.Info().Msgf("Getting commits in range: %s", rangeArg)

	commits, err := s.git.getCommitsInRange(rangeArg, maxCommits)
	if err != nil {
		return nil, err
	}

	detailed, err := s.git.GetCommitDetails(commits)
	if err != nil {
		return nil, err
	}

	return detailed, nil
}

func bumpHigher(a, b string) bool {
	levels := map[string]int{"none": 0, "patch": 1, "minor": 2, "major": 3}
	return levels[a] > levels[b]
}
