package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/releaseforge/helpers"
	"github.com/AxeForging/releaseforge/services"
	"github.com/urfave/cli"
)

type BumpAction struct {
	semverSvc *services.SemverService
	gitSvc    *services.GitService
}

func NewBumpAction(semverSvc *services.SemverService, gitSvc *services.GitService) *BumpAction {
	return &BumpAction{
		semverSvc: semverSvc,
		gitSvc:    gitSvc,
	}
}

func (a *BumpAction) Execute(c *cli.Context) error {
	tag := c.String("tag")
	branch := c.String("branch")
	maxCommits := c.Int("max-commits")
	outputJSON := c.String("output-json")
	outputVersion := c.String("output-version")
	quiet := c.Bool("quiet")

	if tag == "" {
		// Auto-detect latest tag
		latest, err := a.gitSvc.GetLatestPromotedReleaseTag()
		if err != nil || latest == "" {
			return fmt.Errorf("no --tag provided and could not auto-detect latest tag. Use --tag <tag>")
		}
		tag = latest
		if !quiet {
			helpers.Log.Info().Msgf("Auto-detected base tag: %s", tag)
		}
	}

	if !a.gitSvc.IsValidSemverTag(tag) {
		return fmt.Errorf("tag %q is not a valid semver tag", tag)
	}

	if branch == "" {
		branch = "HEAD"
	}

	if maxCommits == 0 {
		maxCommits = 200
	}

	// Get commits between tag and branch
	detailed, err := a.semverSvc.GetCommitsBetween(tag, branch, maxCommits)
	if err != nil {
		return fmt.Errorf("failed to get commits between %s and %s: %w", tag, branch, err)
	}

	if len(detailed) == 0 {
		if !quiet {
			fmt.Printf("No commits found between %s and %s\n", tag, branch)
		}
		return nil
	}

	// Analyze commits
	analysis := a.semverSvc.AnalyzeCommits(detailed)

	// Calculate next version
	nextVersion, err := a.semverSvc.CalculateNextVersion(tag, analysis)
	if err != nil {
		return fmt.Errorf("failed to calculate next version: %w", err)
	}

	// Output
	if quiet {
		fmt.Print(nextVersion)
		return nil
	}

	report := a.semverSvc.FormatAnalysisReport(tag, nextVersion, analysis)
	fmt.Println(report)

	// Save JSON output
	if outputJSON != "" {
		data := map[string]interface{}{
			"base_version":    tag,
			"next_version":    nextVersion,
			"bump_level":      analysis.BumpLevel,
			"total_commits":   analysis.TotalCount,
			"non_conventional": analysis.NonConventionalCount,
			"commits":         analysis.Commits,
		}
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		if err := os.WriteFile(outputJSON, jsonBytes, 0o644); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}
		helpers.Log.Info().Msgf("JSON analysis saved: %s", outputJSON)
	}

	// Save version file
	if outputVersion != "" {
		if err := os.WriteFile(outputVersion, []byte(nextVersion), 0o644); err != nil {
			return fmt.Errorf("failed to write version output: %w", err)
		}
		helpers.Log.Info().Msgf("Version saved: %s (%s)", outputVersion, nextVersion)
	}

	return nil
}

// ExecuteCheck validates that all commits follow conventional commit format
func (a *BumpAction) ExecuteCheck(c *cli.Context) error {
	tag := c.String("tag")
	branch := c.String("branch")
	maxCommits := c.Int("max-commits")
	strict := c.Bool("strict")

	if tag == "" {
		latest, err := a.gitSvc.GetLatestPromotedReleaseTag()
		if err != nil || latest == "" {
			return fmt.Errorf("no --tag provided and could not auto-detect latest tag")
		}
		tag = latest
	}

	if branch == "" {
		branch = "HEAD"
	}
	if maxCommits == 0 {
		maxCommits = 200
	}

	detailed, err := a.semverSvc.GetCommitsBetween(tag, branch, maxCommits)
	if err != nil {
		return err
	}

	analysis := a.semverSvc.AnalyzeCommits(detailed)

	fmt.Println("Conventional Commit Check")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("  Commits analyzed:    %d\n", analysis.TotalCount)
	fmt.Printf("  Conventional:        %d\n", analysis.TotalCount-analysis.NonConventionalCount)
	fmt.Printf("  Non-conventional:    %d\n", analysis.NonConventionalCount)
	fmt.Println()

	if analysis.NonConventionalCount > 0 {
		fmt.Println("  Non-conventional commits:")
		for _, commits := range analysis.Commits {
			for _, c := range commits {
				if !c.Conventional {
					fmt.Printf("    - %s: %s\n", c.Hash, c.Message)
				}
			}
		}

		if strict {
			return fmt.Errorf("found %d non-conventional commits (strict mode)", analysis.NonConventionalCount)
		}
	} else {
		fmt.Println("  All commits follow conventional commit format!")
	}

	return nil
}
