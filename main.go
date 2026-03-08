package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/releaseforge/actions"
	"github.com/AxeForging/releaseforge/helpers"
	"github.com/AxeForging/releaseforge/services"
	"github.com/urfave/cli"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Services
	gitSvc := &services.GitService{}
	llmSvc := services.NewLLMService()
	promptSvc := &services.PromptService{}

	semverSvc := services.NewSemverService(gitSvc)

	// Actions
	generateAction := actions.NewGenerateAction(gitSvc, llmSvc, promptSvc)
	bumpAction := actions.NewBumpAction(semverSvc, gitSvc)

	app := cli.NewApp()
	app.Name = "releaseforge"
	app.Usage = "AI-powered release notes generator with multi-provider LLM support"
	app.Version = Version

	generateFlags := []cli.Flag{
		providerFlag,
		modelFlag,
		keyFlag,
		systemPromptFlag,
		ignoreListFlag,
		templateFlag,
		templateNameFlag,
		templateRawFlag,
		gitShaFlag,
		gitTagFlag,
		analyzeFromTagFlag,
		maxCommitsFlag,
		tagsContextCountFlag,
		disableTagsContextFlag,
		outputFlag,
		useGitFallbackFlag,
		forceGitModeFlag,
		githubTokenFlag,
		verboseFlag,
	}

	app.Commands = []cli.Command{
		{
			Name:    "generate",
			Aliases: []string{"gen", "g"},
			Usage:   "Generate release notes from git commits (using AI or git fallback)",
			Flags:   generateFlags,
			Before: func(c *cli.Context) error {
				if c.Bool("verbose") {
					helpers.SetupLogger("debug")
				}
				return nil
			},
			Action: generateAction.Execute,
		},
		{
			Name:    "templates",
			Aliases: []string{"tpl"},
			Usage:   "List available built-in templates",
			Action: func(c *cli.Context) error {
				fmt.Println("Available built-in templates:")
				fmt.Println()
				for name, content := range services.BuiltinTemplates {
					lines := strings.Split(content, "\n")
					preview := lines[0]
					if len(lines) > 1 {
						for _, l := range lines[1:] {
							l = strings.TrimSpace(l)
							if l != "" {
								preview = l
								break
							}
						}
					}
					fmt.Printf("  %-30s %s\n", name, preview)
				}
				fmt.Println()
				fmt.Println("Use with: releaseforge generate --template-name <name>")
				return nil
			},
		},
		{
			Name:    "bump",
			Aliases: []string{"b"},
			Usage:   "Analyze conventional commits and suggest the next semver version",
			Description: `Analyzes commits between a base tag and a branch/ref using the
Conventional Commits specification to determine the appropriate semver bump.

Commit types and their semver impact:
  MAJOR (breaking changes):
    Any commit with "!" after type/scope (e.g. "feat!:", "fix(api)!:")
    Any commit with "BREAKING CHANGE:" in the footer

  MINOR (new features):
    feat:     A new feature

  PATCH (bug fixes):
    fix:      A bug fix
    perf:     A performance improvement
    revert:   Reverts a previous commit

  NO BUMP (non-functional):
    docs:     Documentation only changes
    style:    Code style changes (formatting, whitespace)
    refactor: Code refactoring (no bug fix or feature)
    test:     Adding or correcting tests
    build:    Build system or external dependency changes
    ci:       CI/CD configuration changes
    chore:    Maintenance tasks

Examples:
  releaseforge bump --tag v1.2.3
  releaseforge bump --tag v1.2.3 --branch main --quiet
  releaseforge bump --output-version next-version.txt --output-json analysis.json`,
			Flags: []cli.Flag{
				bumpTagFlag,
				bumpBranchFlag,
				bumpMaxCommitsFlag,
				bumpOutputJSONFlag,
				bumpOutputVersionFlag,
				bumpQuietFlag,
				verboseFlag,
			},
			Before: func(c *cli.Context) error {
				if c.Bool("verbose") {
					helpers.SetupLogger("debug")
				}
				return nil
			},
			Action: bumpAction.Execute,
		},
		{
			Name:  "check",
			Usage: "Validate that commits follow the Conventional Commits format",
			Description: `Checks all commits between a base tag and a branch/ref to verify
they follow the Conventional Commits specification.

Use --strict to fail with a non-zero exit code if any non-conventional
commits are found (useful in CI pipelines).

Examples:
  releaseforge check --tag v1.2.3
  releaseforge check --tag v1.2.3 --branch main --strict`,
			Flags: []cli.Flag{
				bumpTagFlag,
				bumpBranchFlag,
				bumpMaxCommitsFlag,
				bumpStrictFlag,
				verboseFlag,
			},
			Before: func(c *cli.Context) error {
				if c.Bool("verbose") {
					helpers.SetupLogger("debug")
				}
				return nil
			},
			Action: bumpAction.ExecuteCheck,
		},
		{
			Name:  "version",
			Usage: "Show version information",
			Action: func(c *cli.Context) error {
				fmt.Printf("%s version %s\n", app.Name, Version)
				fmt.Printf("Build time: %s\n", BuildTime)
				fmt.Printf("Git commit: %s\n", GitCommit)
				return nil
			},
		},
	}

	// Default command: generate (when no subcommand specified, run generate)
	app.Action = func(c *cli.Context) error {
		// If args look like flags, treat as generate command
		if c.NArg() == 0 {
			fmt.Println("Usage: releaseforge <command> [options]")
			fmt.Println()
			fmt.Println("Commands:")
			fmt.Println("  generate (gen, g)  Generate release notes from git commits")
			fmt.Println("  bump (b)           Analyze conventional commits and suggest next semver version")
			fmt.Println("  check              Validate commits follow Conventional Commits format")
			fmt.Println("  templates (tpl)    List available built-in templates")
			fmt.Println("  version            Show version information")
			fmt.Println()
			fmt.Println("Use 'releaseforge <command> --help' for more information about a command.")
			return nil
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		helpers.Log.Fatal().Msgf("Fatal error: %v", err)
	}
}
