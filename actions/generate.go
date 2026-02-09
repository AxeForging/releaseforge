package actions

import (
	"fmt"
	"strings"
	"time"

	"github.com/AxeForging/releaseforge/domain"
	"github.com/AxeForging/releaseforge/helpers"
	"github.com/AxeForging/releaseforge/services"
	"github.com/urfave/cli"
)

type GenerateAction struct {
	gitSvc    *services.GitService
	llmSvc    *services.LLMService
	promptSvc *services.PromptService
}

func NewGenerateAction(gitSvc *services.GitService, llmSvc *services.LLMService, promptSvc *services.PromptService) *GenerateAction {
	return &GenerateAction{
		gitSvc:    gitSvc,
		llmSvc:    llmSvc,
		promptSvc: promptSvc,
	}
}

func (a *GenerateAction) Execute(c *cli.Context) error {
	fmt.Println("AI Release Notes - Commit Analysis Tool")
	fmt.Println(strings.Repeat("=", 50))

	inputs := a.parseInputs(c)

	// Resolve template
	template, err := a.resolveTemplate(inputs)
	if err != nil {
		return err
	}

	// Get commits
	commits, isTagAnalysis, tagInfo, err := a.resolveCommits(inputs)
	if err != nil {
		return err
	}

	// Get detailed commits
	var detailedCommits []domain.DetailedCommit
	if len(commits) > 0 {
		detailedCommits, err = a.gitSvc.GetCommitDetails(commits)
		if err != nil {
			helpers.Log.Warn().Msgf("Could not get commit details: %v", err)
		}
	}

	if len(commits) == 0 && len(detailedCommits) == 0 {
		helpers.Log.Warn().Msg("No commits found to analyze")
	}

	// Get existing tags for context
	var existingTags []string
	if !inputs.DisableTagsContext {
		tagsCount := inputs.TagsContextCount
		if tagsCount == 0 {
			tagsCount = 15
		}
		existingTags, _ = a.gitSvc.GetRecentTags(tagsCount)
	}

	// Generate notes
	var result *domain.StructuredResult

	if inputs.ForceGitMode {
		helpers.Log.Info().Msg("Force Git Mode enabled. Skipping LLM generation.")
		result = a.promptSvc.GenerateGitFallbackNotes(detailedCommits, tagInfo, existingTags, a.gitSvc)
	} else {
		result, err = a.generateWithLLM(inputs, commits, template, detailedCommits, isTagAnalysis, tagInfo, existingTags)
		if err != nil {
			if inputs.UseGitFallback {
				helpers.Log.Error().Msgf("LLM generation failed: %v", err)
				helpers.Log.Info().Msg("Falling back to git log analysis...")
				result = a.promptSvc.GenerateGitFallbackNotes(detailedCommits, tagInfo, existingTags, a.gitSvc)
			} else {
				return fmt.Errorf("LLM generation failed: %w", err)
			}
		}
	}

	// Save output
	outputCfg, err := a.promptSvc.SaveStructuredOutput(result, inputs.OutputFile, existingTags)
	if err != nil {
		return fmt.Errorf("failed to save output: %w", err)
	}

	// Print summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Release notes generated successfully!")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("  Markdown: %s\n", outputCfg.MarkdownFile)
	fmt.Printf("  JSON:     %s\n", outputCfg.JSONFile)
	if result.SuggestedVersion != "" {
		fmt.Printf("  Version:  %s (%s)\n", outputCfg.VersionFile, result.SuggestedVersion)
	}
	fmt.Println(strings.Repeat("-", 40))

	return nil
}

func (a *GenerateAction) parseInputs(c *cli.Context) domain.ActionInputs {
	systemPrompt := c.StringSlice("system-prompt")
	if len(systemPrompt) == 0 {
		systemPrompt = []string{"You are a release notes generator. Analyze the git commits and generate comprehensive release notes."}
	}

	maxCommits := c.Int("max-commits")
	if maxCommits == 0 {
		maxCommits = 100
	}

	return domain.ActionInputs{
		Provider:           c.String("provider"),
		Model:              c.String("model"),
		Key:                c.String("key"),
		SystemPrompt:       systemPrompt,
		IgnoreList:         c.StringSlice("ignore-list"),
		Template:           c.String("template"),
		TemplateName:       c.String("template-name"),
		TemplateRaw:        c.String("template-raw"),
		GitSha:             c.String("git-sha"),
		GitTag:             c.String("git-tag"),
		AnalyzeFromTag:     c.Bool("analyze-from-tag"),
		MaxCommits:         maxCommits,
		TagsContextCount:   c.Int("tags-context-count"),
		DisableTagsContext: c.Bool("disable-tags-context"),
		OutputFile:         c.String("output"),
		UseGitFallback:     c.Bool("use-git-fallback"),
		ForceGitMode:       c.Bool("force-git-mode"),
	}
}

func (a *GenerateAction) resolveTemplate(inputs domain.ActionInputs) (string, error) {
	if inputs.TemplateRaw != "" {
		helpers.Log.Info().Msg("Using template-raw input as template content")
		return inputs.TemplateRaw, nil
	}

	if inputs.TemplateName != "" {
		t, err := a.promptSvc.GetTemplate(inputs.TemplateName)
		if err != nil {
			return "", err
		}
		helpers.Log.Info().Msgf("Using built-in template: %s", inputs.TemplateName)
		return t, nil
	}

	if inputs.Template != "" {
		t, err := a.promptSvc.ReadTemplate(inputs.Template)
		if err != nil {
			return "", err
		}
		helpers.Log.Info().Msgf("Using custom template file: %s", inputs.Template)
		return t, nil
	}

	helpers.Log.Info().Msg("No template provided. Using default: semver-release-notes")
	t, _ := a.promptSvc.GetTemplate("semver-release-notes")
	return t, nil
}

func (a *GenerateAction) resolveCommits(inputs domain.ActionInputs) ([]domain.CommitInfo, bool, *domain.TagInfo, error) {
	var isTagAnalysis bool
	var tagInfo *domain.TagInfo

	if inputs.GitTag != "" {
		isTagAnalysis = true
		tagInfo = &domain.TagInfo{
			CurrentTag:  inputs.GitTag,
			ReleaseDate: time.Now().Format("2006-01-02"),
		}
		if inputs.AnalyzeFromTag {
			prevTag, _ := a.gitSvc.GetPreviousTag(inputs.GitTag)
			if prevTag != "" {
				tagInfo.PreviousTag = prevTag
			}
		}
	} else if inputs.GitSha == "" {
		latestTag, err := a.gitSvc.GetLatestPromotedReleaseTag()
		if err == nil && latestTag != "" {
			helpers.Log.Info().Msgf("Auto-detected latest promoted release: %s", latestTag)
			isTagAnalysis = true
			tagInfo = &domain.TagInfo{
				CurrentTag:  latestTag,
				ReleaseDate: time.Now().Format("2006-01-02"),
			}
		}
	}

	opts := domain.GitAnalysisOptions{
		IgnoreList:     inputs.IgnoreList,
		MaxCommits:     inputs.MaxCommits,
		GitSha:         inputs.GitSha,
		GitTag:         inputs.GitTag,
		AnalyzeFromTag: inputs.AnalyzeFromTag,
	}

	commits, err := a.gitSvc.GetFilteredCommits(opts)
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to get commits: %w", err)
	}

	return commits, isTagAnalysis, tagInfo, nil
}

func (a *GenerateAction) generateWithLLM(
	inputs domain.ActionInputs,
	commits []domain.CommitInfo,
	template string,
	detailedCommits []domain.DetailedCommit,
	isTagAnalysis bool,
	tagInfo *domain.TagInfo,
	existingTags []string,
) (*domain.StructuredResult, error) {
	if inputs.Key == "" {
		return nil, fmt.Errorf("API key is required for LLM generation. Use --key or --force-git-mode")
	}

	prompt := a.promptSvc.BuildPrompt(
		inputs.SystemPrompt,
		commits,
		template,
		detailedCommits,
		isTagAnalysis,
		tagInfo,
		existingTags,
	)

	helpers.Log.Info().Msgf("Sending prompt to %s (%s)...", inputs.Provider, inputs.Model)
	start := time.Now()

	response, err := a.llmSvc.Generate(inputs.Provider, inputs.Key, inputs.Model, prompt)
	if err != nil {
		return nil, err
	}

	duration := time.Since(start).Seconds()
	helpers.Log.Info().Msgf("LLM response received in %.2fs (%d characters)", duration, len(response))

	return a.promptSvc.ParseResponse(response)
}
