package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/AxeForging/releasenotes/domain"
	"github.com/AxeForging/releasenotes/helpers"
)

type PromptService struct{}

func (p *PromptService) ReadTemplate(path string) (string, error) {
	helpers.Log.Info().Msgf("Reading template from: %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", helpers.WrapError(err, "template", fmt.Sprintf("failed to read template file %q", path))
	}
	helpers.Log.Info().Msgf("Template loaded successfully (%d characters)", len(data))
	return string(data), nil
}

func (p *PromptService) BuildPrompt(
	systemPrompts []string,
	commits []domain.CommitInfo,
	template string,
	detailedCommits []domain.DetailedCommit,
	isTagAnalysis bool,
	tagInfo *domain.TagInfo,
	existingTags []string,
) string {
	var sb strings.Builder

	// System prompt
	if len(systemPrompts) > 0 {
		sb.WriteString(strings.Join(systemPrompts, "\n"))
		sb.WriteString("\n\n")
	}

	sb.WriteString("First, analyze the following commits to generate release notes.\n")

	// Tag context
	if isTagAnalysis && tagInfo != nil {
		sb.WriteString("\n## Release Information\n")
		if tagInfo.CurrentTag != "" {
			sb.WriteString(fmt.Sprintf("- **Current Tag**: %s\n", tagInfo.CurrentTag))
		}
		sb.WriteString(fmt.Sprintf("- **Release Date**: %s\n", tagInfo.ReleaseDate))
		if tagInfo.PreviousTag != "" {
			sb.WriteString(fmt.Sprintf("- **Previous Tag**: %s\n", tagInfo.PreviousTag))
		}
		sb.WriteString("\n## Commits to Analyze\n")
	}

	// Existing tags context
	if len(existingTags) > 0 {
		sb.WriteString("\n## Existing Tags Context\n")
		sb.WriteString("Here are the existing tags in this repository for context:\n")
		for _, tag := range existingTags {
			sb.WriteString(fmt.Sprintf("- %s\n", tag))
		}

		// Analyze versioning pattern
		var semverTags []string
		gitSvc := &GitService{}
		for _, tag := range existingTags {
			if gitSvc.IsValidSemverTag(tag) {
				semverTags = append(semverTags, tag)
			}
		}

		if len(semverTags) > 0 {
			sb.WriteString(fmt.Sprintf("\n**Versioning Analysis:**\n- Latest semantic version: %s\n- Total semantic versions found: %d\n", semverTags[0], len(semverTags)))
		}

		if isTagAnalysis {
			sb.WriteString("\nWhen suggesting a new version, consider the existing versioning pattern and ensure your suggestion follows the same format and progression.\n")
			sb.WriteString("\nIMPORTANT: Do NOT suggest a version that already exists in the repository.\n")
		}
		sb.WriteString("\n")
	}

	// Commit lines
	sb.WriteString("\nHere are the commits to analyze:\n")
	if len(detailedCommits) > 0 {
		for _, c := range detailedCommits {
			fileInfo := ""
			if len(c.FilesChanged) > 0 {
				showing := c.FilesChanged
				if len(showing) > 3 {
					showing = showing[:3]
				}
				fileInfo = fmt.Sprintf(" [%d files: %s", c.FileCount, strings.Join(showing, ", "))
				if c.FileCount > 3 {
					fileInfo += "..."
				}
				fileInfo += "]"
			}
			authorInfo := ""
			if c.Author != "" {
				authorInfo = fmt.Sprintf(" (by %s)", c.Author)
			}
			sb.WriteString(fmt.Sprintf("- %s: %s%s%s\n", shortHash(c.Hash), firstLine(c.Message), fileInfo, authorInfo))
		}
	} else {
		for _, c := range commits {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", c.Hash, c.Message))
		}
	}

	// Template
	sb.WriteString(fmt.Sprintf("\nUse this markdown template to structure the 'release_notes' value:\n```markdown\n%s\n```\n", template))

	// Version suggestion instructions
	if isTagAnalysis && tagInfo != nil {
		sb.WriteString(fmt.Sprintf(`
Based on the commits, determine the next semantic version for the 'suggested_version' field.
The current version is **%s**. Please suggest the next version by incrementing the version according to Semantic Versioning rules:
- Increment **MAJOR** for breaking changes (API changes, incompatible changes).
- Increment **MINOR** for new features (backward-compatible new functionality).
- Increment **PATCH** for bug fixes (backward-compatible bug fixes).

Look for keywords in commit messages like:
- Breaking changes: "breaking", "BREAKING", "major", "incompatible"
- New features: "feat", "feature", "add", "new", "enhancement"
- Bug fixes: "fix", "bug", "patch", "resolve", "correct"

The value for 'suggested_version' must be a valid version string (e.g., "1.2.4") and nothing else.
`, tagInfo.CurrentTag))
	}

	// JSON output format
	sb.WriteString(`
Finally, format your entire response as a JSON object with exactly these fields:
{
  "release_notes": "The complete release notes formatted in markdown based on the template above",
  "suggested_version": "The suggested next semantic version (e.g., '1.2.3') or null if no version can be suggested"
}

IMPORTANT: Your response must be ONLY the JSON object, no markdown code fences, no extra text before or after the JSON.
`)

	return sb.String()
}

func (p *PromptService) ParseResponse(text string) (*domain.StructuredResult, error) {
	helpers.Log.Info().Msg("Parsing LLM response...")

	// Strip markdown code fences if present
	cleaned := text
	cleaned = strings.TrimSpace(cleaned)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	}
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
	}
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)

	var result domain.StructuredResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// Try to extract JSON from the text
		jsonStart := strings.Index(text, "{")
		jsonEnd := strings.LastIndex(text, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			extracted := text[jsonStart : jsonEnd+1]
			if err2 := json.Unmarshal([]byte(extracted), &result); err2 != nil {
				return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nRaw output: %s", err, text[:min(500, len(text))])
			}
		} else {
			return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nRaw output: %s", err, text[:min(500, len(text))])
		}
	}

	helpers.Log.Info().Msg("Structured JSON parsed successfully")
	return &result, nil
}

func (p *PromptService) GenerateOutputPaths(basePath string) domain.OutputConfig {
	if basePath == "" {
		return domain.OutputConfig{
			MarkdownFile: "/tmp/releasenotes-output.md",
			VersionFile:  "/tmp/suggested-version.txt",
			JSONFile:     "/tmp/releasenotes-output.json",
		}
	}

	hasSep := strings.Contains(basePath, "/") || strings.Contains(basePath, "\\")
	if !hasSep {
		return domain.OutputConfig{
			MarkdownFile: "./" + basePath + ".md",
			VersionFile:  "./suggested-version.txt",
			JSONFile:     "./" + basePath + ".json",
		}
	}

	outputDir := basePath
	baseFileName := "releasenotes-output"

	ext := filepath.Ext(basePath)
	if ext == ".md" || ext == ".json" || ext == ".txt" {
		outputDir = filepath.Dir(basePath)
		baseFileName = strings.TrimSuffix(filepath.Base(basePath), ext)
	} else if strings.HasSuffix(basePath, "/") {
		outputDir = basePath
	}

	return domain.OutputConfig{
		MarkdownFile: filepath.Join(outputDir, baseFileName+".md"),
		VersionFile:  filepath.Join(outputDir, "suggested-version.txt"),
		JSONFile:     filepath.Join(outputDir, baseFileName+".json"),
	}
}

func (p *PromptService) SaveStructuredOutput(result *domain.StructuredResult, outputPath string, existingTags []string) (*domain.OutputConfig, error) {
	helpers.Log.Info().Msg("Processing structured output...")

	cfg := p.GenerateOutputPaths(outputPath)

	// Ensure output directory exists
	dir := filepath.Dir(cfg.MarkdownFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, helpers.WrapError(err, "output", "failed to create output directory")
		}
	}

	// Save markdown
	if err := os.WriteFile(cfg.MarkdownFile, []byte(result.ReleaseNotes), 0o644); err != nil {
		return nil, helpers.WrapError(err, "output", "failed to write markdown file")
	}
	helpers.Log.Info().Msgf("Markdown saved: %s", cfg.MarkdownFile)

	// Save JSON
	jsonData := domain.OutputJSON{
		Content:          result.ReleaseNotes,
		SuggestedVersion: result.SuggestedVersion,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return nil, helpers.WrapError(err, "output", "failed to marshal JSON")
	}
	if err := os.WriteFile(cfg.JSONFile, jsonBytes, 0o644); err != nil {
		return nil, helpers.WrapError(err, "output", "failed to write JSON file")
	}
	helpers.Log.Info().Msgf("JSON saved: %s", cfg.JSONFile)

	// Save version file
	if result.SuggestedVersion != "" {
		if versionAlreadyExists(result.SuggestedVersion, existingTags) {
			helpers.Log.Warn().Msgf("Suggested version '%s' already exists in repository. Skipping version file creation.", result.SuggestedVersion)
		} else {
			if err := os.WriteFile(cfg.VersionFile, []byte(result.SuggestedVersion), 0o644); err != nil {
				return nil, helpers.WrapError(err, "output", "failed to write version file")
			}
			helpers.Log.Info().Msgf("Version file saved: %s (%s)", cfg.VersionFile, result.SuggestedVersion)
		}
	}

	return &cfg, nil
}

func versionAlreadyExists(suggested string, existingTags []string) bool {
	if len(existingTags) == 0 {
		return false
	}

	normalized := strings.TrimSpace(suggested)
	withoutV := strings.TrimPrefix(normalized, "v")
	withV := "v" + withoutV

	for _, tag := range existingTags {
		tag = strings.TrimSpace(tag)
		tagWithoutV := strings.TrimPrefix(tag, "v")
		tagWithV := "v" + tagWithoutV
		if tag == normalized || tagWithoutV == withoutV || tagWithV == withV {
			return true
		}
	}
	return false
}

func shortHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}

func (p *PromptService) GetTemplate(name string) (string, error) {
	t, ok := BuiltinTemplates[name]
	if !ok {
		available := make([]string, 0, len(BuiltinTemplates))
		for k := range BuiltinTemplates {
			available = append(available, k)
		}
		return "", fmt.Errorf("template %q not found. Available: %s", name, strings.Join(available, ", "))
	}
	return t, nil
}

func (p *PromptService) AvailableTemplates() []string {
	names := make([]string, 0, len(BuiltinTemplates))
	for k := range BuiltinTemplates {
		names = append(names, k)
	}
	return names
}

func (p *PromptService) GenerateGitFallbackNotes(
	detailedCommits []domain.DetailedCommit,
	tagInfo *domain.TagInfo,
	existingTags []string,
	gitSvc *GitService,
) *domain.StructuredResult {
	helpers.Log.Info().Msg("Generating fallback notes from git history...")

	var notes strings.Builder
	notes.WriteString("# Release Notes (Git Log Fallback)\n\n")

	if tagInfo != nil {
		notes.WriteString("## Release Information\n")
		if tagInfo.CurrentTag != "" {
			notes.WriteString(fmt.Sprintf("- **Current Tag**: %s\n", tagInfo.CurrentTag))
		}
		notes.WriteString(fmt.Sprintf("- **Release Date**: %s\n", tagInfo.ReleaseDate))
		if tagInfo.PreviousTag != "" {
			notes.WriteString(fmt.Sprintf("- **Previous Tag**: %s\n", tagInfo.PreviousTag))
		}
		notes.WriteString("\n")
	}

	notes.WriteString("## Commits\n\n")

	if len(detailedCommits) == 0 {
		notes.WriteString("No commits found.\n")
	} else {
		// Categorize commits by conventional commit type
		categories := categorizeCommits(detailedCommits)
		if len(categories["feat"]) > 0 {
			notes.WriteString("### Features\n")
			for _, c := range categories["feat"] {
				notes.WriteString(fmt.Sprintf("- %s: %s (by %s)\n", shortHash(c.Hash), firstLine(c.Message), c.Author))
			}
			notes.WriteString("\n")
		}
		if len(categories["fix"]) > 0 {
			notes.WriteString("### Fixes\n")
			for _, c := range categories["fix"] {
				notes.WriteString(fmt.Sprintf("- %s: %s (by %s)\n", shortHash(c.Hash), firstLine(c.Message), c.Author))
			}
			notes.WriteString("\n")
		}
		if len(categories["other"]) > 0 {
			notes.WriteString("### Other Changes\n")
			for _, c := range categories["other"] {
				notes.WriteString(fmt.Sprintf("- %s: %s (by %s)\n", shortHash(c.Hash), firstLine(c.Message), c.Author))
			}
			notes.WriteString("\n")
		}

		// Contributors
		notes.WriteString("## Contributors\n\n")
		authors := uniqueAuthors(detailedCommits)
		if len(authors) > 0 {
			for _, a := range authors {
				notes.WriteString(fmt.Sprintf("- %s\n", a))
			}
		} else {
			notes.WriteString("- N/A\n")
		}
	}

	// Calculate suggested version
	var suggestedVersion string
	if tagInfo != nil && tagInfo.CurrentTag != "" {
		parsed := gitSvc.ParseSemverTag(tagInfo.CurrentTag)
		if parsed != nil {
			suggestedVersion = calcNextVersion(parsed, detailedCommits, tagInfo.CurrentTag)
		}
	} else if len(existingTags) > 0 {
		parsed := gitSvc.ParseSemverTag(existingTags[0])
		if parsed != nil {
			parsed.Patch++
			prefix := ""
			if strings.HasPrefix(existingTags[0], "v") {
				prefix = "v"
			}
			suggestedVersion = fmt.Sprintf("%s%d.%d.%d", prefix, parsed.Major, parsed.Minor, parsed.Patch)
		}
	}

	if suggestedVersion == "" && len(existingTags) == 0 {
		suggestedVersion = "0.0.1"
	}

	return &domain.StructuredResult{
		ReleaseNotes:     notes.String(),
		SuggestedVersion: suggestedVersion,
	}
}

func categorizeCommits(commits []domain.DetailedCommit) map[string][]domain.DetailedCommit {
	cats := map[string][]domain.DetailedCommit{
		"feat":  {},
		"fix":   {},
		"other": {},
	}

	featRe := regexp.MustCompile(`(?i)^(feat|feature)[\(:]`)
	fixRe := regexp.MustCompile(`(?i)^(fix|bugfix)[\(:]`)

	for _, c := range commits {
		msg := firstLine(c.Message)
		switch {
		case featRe.MatchString(msg):
			cats["feat"] = append(cats["feat"], c)
		case fixRe.MatchString(msg):
			cats["fix"] = append(cats["fix"], c)
		default:
			cats["other"] = append(cats["other"], c)
		}
	}
	return cats
}

func uniqueAuthors(commits []domain.DetailedCommit) []string {
	seen := map[string]bool{}
	var authors []string
	for _, c := range commits {
		if c.Author != "" && !seen[c.Author] {
			seen[c.Author] = true
			authors = append(authors, c.Author)
		}
	}
	return authors
}

func calcNextVersion(parsed *domain.SemVer, commits []domain.DetailedCommit, currentTag string) string {
	increment := "patch"
	for _, c := range commits {
		msg := strings.ToLower(c.Message)
		if strings.Contains(msg, "breaking") || strings.Contains(msg, "major") {
			increment = "major"
			break
		} else if strings.Contains(msg, "feat") || strings.Contains(msg, "feature") {
			if increment == "patch" {
				increment = "minor"
			}
		}
	}

	next := *parsed
	switch increment {
	case "major":
		next.Major++
		next.Minor = 0
		next.Patch = 0
	case "minor":
		next.Minor++
		next.Patch = 0
	default:
		next.Patch++
	}

	prefix := ""
	if strings.HasPrefix(currentTag, "v") {
		prefix = "v"
	}
	return fmt.Sprintf("%s%d.%d.%d", prefix, next.Major, next.Minor, next.Patch)
}
