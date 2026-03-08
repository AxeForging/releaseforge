package services

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/AxeForging/releaseforge/domain"
	"github.com/AxeForging/releaseforge/helpers"
)

type GitService struct{}

func (g *GitService) GetFilteredCommits(opts domain.GitAnalysisOptions) ([]domain.CommitInfo, error) {
	if opts.GitSha != "" {
		helpers.Log.Info().Msgf("Analyzing specific commit: %s", opts.GitSha)
		return g.analyzeSingleCommit(opts.GitSha, opts.IgnoreList)
	}

	if opts.GitTag != "" && opts.AnalyzeFromTag {
		helpers.Log.Info().Msgf("Analyzing all commits after tag: %s", opts.GitTag)
		return g.getCommitsAfterTag(opts.GitTag, opts.IgnoreList, opts.MaxCommits)
	}

	if opts.GitTag != "" {
		helpers.Log.Info().Msgf("Analyzing commits for tag: %s", opts.GitTag)
		return g.getCommitsForTag(opts.GitTag, opts.IgnoreList, opts.MaxCommits)
	}

	helpers.Log.Info().Msg("No specific commit or tag provided, checking for latest promoted release...")
	latestTag, err := g.GetLatestPromotedReleaseTag()
	if err == nil && latestTag != "" {
		helpers.Log.Info().Msgf("Found latest promoted release: %s. Analyzing commits since this release...", latestTag)
		return g.getCommitsAfterTag(latestTag, opts.IgnoreList, opts.MaxCommits)
	}

	helpers.Log.Info().Msgf("No tags found, fetching recent commits (max: %d)...", opts.MaxCommits)
	return g.analyzeRecentCommits(opts.IgnoreList, opts.MaxCommits)
}

func (g *GitService) analyzeSingleCommit(sha string, ignoreList []string) ([]domain.CommitInfo, error) {
	// Verify commit exists
	out, err := g.runGit("show", sha, "--format=%B", "--no-patch")
	if err != nil {
		return nil, fmt.Errorf("commit %s not found: %w", sha, err)
	}

	// Check ignored files
	filesOut, err := g.runGit("show", sha, "--name-only", "--pretty=format:")
	if err == nil {
		files := splitLines(filesOut)
		if g.touchesIgnored(files, ignoreList) {
			helpers.Log.Info().Msgf("Commit %s touches ignored files - skipping", sha[:8])
			return nil, nil
		}
	}

	return []domain.CommitInfo{{Hash: sha, Message: strings.TrimSpace(out)}}, nil
}

func (g *GitService) getCommitsAfterTag(tag string, ignoreList []string, maxCommits int) ([]domain.CommitInfo, error) {
	format := "%H|||%s"
	out, err := g.runGit("log", fmt.Sprintf("%s..HEAD", tag), fmt.Sprintf("--format=%s", format), fmt.Sprintf("--max-count=%d", maxCommits))
	if err != nil {
		return nil, fmt.Errorf("failed to get commits after tag %s: %w", tag, err)
	}
	return g.parseAndFilterCommits(out, ignoreList)
}

func (g *GitService) getCommitsForTag(tag string, ignoreList []string, maxCommits int) ([]domain.CommitInfo, error) {
	format := "%H|||%s"
	out, err := g.runGit("log", tag, fmt.Sprintf("--format=%s", format), fmt.Sprintf("--max-count=%d", maxCommits))
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for tag %s: %w", tag, err)
	}
	return g.parseAndFilterCommits(out, ignoreList)
}

func (g *GitService) analyzeRecentCommits(ignoreList []string, maxCommits int) ([]domain.CommitInfo, error) {
	format := "%H|||%s"
	out, err := g.runGit("log", fmt.Sprintf("--format=%s", format), fmt.Sprintf("--max-count=%d", maxCommits))
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}
	return g.parseAndFilterCommits(out, ignoreList)
}

func (g *GitService) parseAndFilterCommits(output string, ignoreList []string) ([]domain.CommitInfo, error) {
	lines := splitLines(output)
	var commits []domain.CommitInfo
	ignored := 0

	for _, line := range lines {
		parts := strings.SplitN(line, "|||", 2)
		if len(parts) != 2 {
			continue
		}
		hash := strings.TrimSpace(parts[0])
		msg := strings.TrimSpace(parts[1])

		if len(ignoreList) > 0 {
			filesOut, err := g.runGit("show", hash, "--name-only", "--pretty=format:")
			if err == nil {
				files := splitLines(filesOut)
				if g.touchesIgnored(files, ignoreList) {
					ignored++
					continue
				}
			}
		}

		commits = append(commits, domain.CommitInfo{Hash: hash, Message: msg})
	}

	helpers.Log.Info().Msgf("Filtered commits: %d included, %d ignored", len(commits), ignored)
	return commits, nil
}

func (g *GitService) GetCommitDetails(commits []domain.CommitInfo, ghSvc *GitHubService) ([]domain.DetailedCommit, error) {
	helpers.Log.Info().Msg("Gathering detailed commit information...")
	var detailed []domain.DetailedCommit

	for _, c := range commits {
		dc := domain.DetailedCommit{
			Hash:    c.Hash,
			Message: c.Message,
		}

		// Get author name, email, and date
		authorOut, err := g.runGit("show", c.Hash, "--format=%an", "--no-patch")
		if err == nil {
			dc.Author = strings.TrimSpace(authorOut)
		}

		emailOut, err := g.runGit("show", c.Hash, "--format=%ae", "--no-patch")
		if err == nil {
			dc.AuthorEmail = strings.TrimSpace(emailOut)
			if ghSvc != nil {
				dc.Author = ghSvc.ResolveAuthor(dc.Author, dc.AuthorEmail)
			} else if ghUser := extractGitHubUser(dc.AuthorEmail); ghUser != "" {
				dc.Author = "@" + ghUser
			}
		}

		dateOut, err := g.runGit("show", c.Hash, "--format=%ci", "--no-patch")
		if err == nil {
			dc.Date = strings.TrimSpace(dateOut)
		}

		// Get changed files
		filesOut, err := g.runGit("show", c.Hash, "--name-only", "--pretty=format:")
		if err == nil {
			dc.FilesChanged = splitLines(filesOut)
			dc.FileCount = len(dc.FilesChanged)
		}

		detailed = append(detailed, dc)
	}

	helpers.Log.Info().Msgf("Enhanced %d commits with detailed information", len(detailed))
	return detailed, nil
}

func (g *GitService) GetPreviousTag(currentTag string) (string, error) {
	out, err := g.runGit("tag", "--sort=-version:refname")
	if err != nil {
		return "", err
	}

	tags := splitLines(out)
	for i, tag := range tags {
		if tag == currentTag && i+1 < len(tags) {
			return tags[i+1], nil
		}
	}
	return "", nil
}

func (g *GitService) GetRecentTags(count int) ([]string, error) {
	out, err := g.runGit("tag", "--sort=-version:refname")
	if err != nil {
		return nil, err
	}

	tags := splitLines(out)
	if len(tags) > count {
		tags = tags[:count]
	}

	helpers.Log.Info().Msgf("Retrieved %d recent tags for context", len(tags))
	return tags, nil
}

func (g *GitService) GetLatestPromotedReleaseTag() (string, error) {
	out, err := g.runGit("tag", "--sort=-version:refname")
	if err != nil {
		return "", err
	}

	tags := splitLines(out)
	if len(tags) == 0 {
		return "", nil
	}
	return tags[0], nil
}

func (g *GitService) ParseSemverTag(tag string) *domain.SemVer {
	version := tag
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}

	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	match := re.FindStringSubmatch(version)
	if match == nil {
		return nil
	}

	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])

	return &domain.SemVer{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: match[4],
		Build:      match[5],
	}
}

func (g *GitService) getCommitsInRange(rangeArg string, maxCommits int) ([]domain.CommitInfo, error) {
	format := "%H|||%B%x00"
	out, err := g.runGit("log", rangeArg, fmt.Sprintf("--format=%s", format), fmt.Sprintf("--max-count=%d", maxCommits))
	if err != nil {
		return nil, fmt.Errorf("failed to get commits in range %s: %w", rangeArg, err)
	}

	var commits []domain.CommitInfo
	entries := strings.Split(out, "\x00")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "|||", 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, domain.CommitInfo{
			Hash:    strings.TrimSpace(parts[0]),
			Message: strings.TrimSpace(parts[1]),
		})
	}

	helpers.Log.Info().Msgf("Found %d commits in range %s", len(commits), rangeArg)
	return commits, nil
}

func (g *GitService) IsValidSemverTag(tag string) bool {
	return g.ParseSemverTag(tag) != nil
}

func (g *GitService) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func (g *GitService) touchesIgnored(files, ignoreList []string) bool {
	for _, f := range files {
		for _, ig := range ignoreList {
			if strings.TrimSpace(f) == strings.TrimSpace(ig) {
				return true
			}
		}
	}
	return false
}

func extractGitHubUser(email string) string {
	if !strings.HasSuffix(email, "@users.noreply.github.com") {
		return ""
	}
	local := strings.TrimSuffix(email, "@users.noreply.github.com")
	// Handle id+username format (e.g. 12345+username@users.noreply.github.com)
	if idx := strings.Index(local, "+"); idx >= 0 {
		return local[idx+1:]
	}
	return local
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
