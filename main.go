package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/releasenotes/actions"
	"github.com/AxeForging/releasenotes/helpers"
	"github.com/AxeForging/releasenotes/services"
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

	// Actions
	generateAction := actions.NewGenerateAction(gitSvc, llmSvc, promptSvc)

	app := cli.NewApp()
	app.Name = "releasenotes"
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
				fmt.Println("Use with: releasenotes generate --template-name <name>")
				return nil
			},
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
			fmt.Println("Usage: releasenotes <command> [options]")
			fmt.Println()
			fmt.Println("Commands:")
			fmt.Println("  generate (gen, g)  Generate release notes from git commits")
			fmt.Println("  templates (tpl)    List available built-in templates")
			fmt.Println("  version            Show version information")
			fmt.Println()
			fmt.Println("Use 'releasenotes <command> --help' for more information about a command.")
			return nil
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		helpers.Log.Fatal().Msgf("Fatal error: %v", err)
	}
}
