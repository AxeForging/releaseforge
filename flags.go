package main

import "github.com/urfave/cli"

var providerFlag = cli.StringFlag{
	Name:   "provider, p",
	Value:  "gemini",
	Usage:  "LLM provider: gemini, openai, or anthropic (default: gemini)",
	EnvVar: "RELEASENOTES_PROVIDER",
}

var modelFlag = cli.StringFlag{
	Name:   "model, m",
	Value:  "gemini-2.0-flash",
	Usage:  "Model name (e.g. gemini-2.0-flash, gpt-4o, claude-sonnet-4-5-20250929)",
	EnvVar: "RELEASENOTES_MODEL",
}

var keyFlag = cli.StringFlag{
	Name:   "key, k",
	Usage:  "API key for the LLM provider",
	EnvVar: "GEMINI_API_KEY,OPENAI_API_KEY,ANTHROPIC_API_KEY",
}

var systemPromptFlag = cli.StringSliceFlag{
	Name:  "system-prompt, sp",
	Usage: "One or more system-level prompt lines",
}

var ignoreListFlag = cli.StringSliceFlag{
	Name:  "ignore-list, il",
	Usage: "File paths to ignore in commit diffs",
}

var templateFlag = cli.StringFlag{
	Name:  "template, t",
	Usage: "Path to a file containing the markdown template",
}

var templateNameFlag = cli.StringFlag{
	Name:  "template-name, tn",
	Value: "",
	Usage: "Built-in template name: semver-release-notes, conventional-changelog, version-analysis",
}

var templateRawFlag = cli.StringFlag{
	Name:  "template-raw, tr",
	Usage: "Raw template content as a string (takes precedence over template and template-name)",
}

var gitShaFlag = cli.StringFlag{
	Name:  "git-sha",
	Usage: "Specific commit SHA to analyze",
}

var gitTagFlag = cli.StringFlag{
	Name:  "git-tag",
	Usage: "Git tag to analyze commits from or for",
}

var analyzeFromTagFlag = cli.BoolFlag{
	Name:  "analyze-from-tag",
	Usage: "Analyze all commits after the specified tag (use with --git-tag)",
}

var maxCommitsFlag = cli.IntFlag{
	Name:  "max-commits",
	Value: 100,
	Usage: "Maximum number of commits to analyze",
}

var tagsContextCountFlag = cli.IntFlag{
	Name:  "tags-context-count",
	Value: 15,
	Usage: "Number of existing tags to include for context",
}

var disableTagsContextFlag = cli.BoolFlag{
	Name:  "disable-tags-context",
	Usage: "Disable fetching existing tags for context",
}

var outputFlag = cli.StringFlag{
	Name:  "output, o",
	Usage: "Custom output file path (default: /tmp)",
}

var useGitFallbackFlag = cli.BoolFlag{
	Name:  "use-git-fallback",
	Usage: "Fallback to git commit log analysis if LLM fails (default: true)",
}

var forceGitModeFlag = cli.BoolFlag{
	Name:  "force-git-mode",
	Usage: "Force using git commit log analysis instead of LLM (no API key needed)",
}

var verboseFlag = cli.BoolFlag{
	Name:  "verbose, v",
	Usage: "Enable verbose/debug logging",
}
