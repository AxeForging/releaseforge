package domain

type CommitInfo struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

type DetailedCommit struct {
	Hash         string   `json:"hash"`
	Message      string   `json:"message"`
	Author       string   `json:"author"`
	AuthorEmail  string   `json:"author_email"`
	Date         string   `json:"date"`
	FilesChanged []string `json:"files_changed"`
	FileCount    int      `json:"file_count"`
}

type GitAnalysisOptions struct {
	IgnoreList     []string
	MaxCommits     int
	GitSha         string
	GitTag         string
	AnalyzeFromTag bool
}

type TagInfo struct {
	CurrentTag  string `json:"current_tag,omitempty"`
	PreviousTag string `json:"previous_tag,omitempty"`
	ReleaseDate string `json:"release_date,omitempty"`
}

type OutputConfig struct {
	MarkdownFile string `json:"markdown_file"`
	VersionFile  string `json:"version_file"`
	JSONFile     string `json:"json_file"`
}

type StructuredResult struct {
	ReleaseNotes     string `json:"release_notes"`
	SuggestedVersion string `json:"suggested_version"`
}

type OutputJSON struct {
	Content          string `json:"content"`
	SuggestedVersion string `json:"suggested_version"`
	GeneratedAt      string `json:"generated_at"`
}

type ActionInputs struct {
	Provider           string
	Model              string
	Key                string
	SystemPrompt       []string
	IgnoreList         []string
	Template           string
	TemplateName       string
	TemplateRaw        string
	GitSha             string
	GitTag             string
	AnalyzeFromTag     bool
	MaxCommits         int
	TagsContextCount   int
	DisableTagsContext bool
	OutputFile         string
	UseGitFallback     bool
	ForceGitMode       bool
}

type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

type AnalyzedCommit struct {
	Hash         string `json:"hash"`
	Message      string `json:"message"`
	Type         string `json:"type"`
	Scope        string `json:"scope,omitempty"`
	Description  string `json:"description"`
	Breaking     bool   `json:"breaking"`
	BumpLevel    string `json:"bump_level"`
	Conventional bool   `json:"conventional"`
}

type BumpAnalysis struct {
	BumpLevel            string                      `json:"bump_level"`
	Commits              map[string][]AnalyzedCommit `json:"commits"`
	TotalCount           int                         `json:"total_count"`
	NonConventionalCount int                         `json:"non_conventional_count"`
}
