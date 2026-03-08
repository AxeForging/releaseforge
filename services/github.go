package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/AxeForging/releaseforge/helpers"
)

type GitHubService struct {
	token string
	cache map[string]string
	mu    sync.Mutex
}

func NewGitHubService(token string) *GitHubService {
	return &GitHubService{
		token: token,
		cache: make(map[string]string),
	}
}

func (gh *GitHubService) HasToken() bool {
	return gh.token != ""
}

func (gh *GitHubService) ResolveUsername(email string) string {
	if email == "" {
		return ""
	}

	// Check noreply format first (no API needed)
	if user := extractGitHubUser(email); user != "" {
		return user
	}

	if !gh.HasToken() {
		return ""
	}

	gh.mu.Lock()
	if cached, ok := gh.cache[email]; ok {
		gh.mu.Unlock()
		return cached
	}
	gh.mu.Unlock()

	username := gh.searchUserByEmail(email)

	gh.mu.Lock()
	gh.cache[email] = username
	gh.mu.Unlock()

	return username
}

func (gh *GitHubService) searchUserByEmail(email string) string {
	url := fmt.Sprintf("https://api.github.com/search/users?q=%s+in:email", email)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+gh.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		helpers.Log.Debug().Msgf("GitHub API request failed for %s: %v", email, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		helpers.Log.Debug().Msgf("GitHub API returned %d for email %s", resp.StatusCode, email)
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Login string `json:"login"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	if result.TotalCount > 0 && len(result.Items) > 0 {
		username := result.Items[0].Login
		helpers.Log.Info().Msgf("Resolved email %s → @%s", email, username)
		return username
	}

	// Try commit search as fallback (works even when email isn't public)
	return gh.searchCommitAuthor(email)
}

func (gh *GitHubService) searchCommitAuthor(email string) string {
	url := fmt.Sprintf("https://api.github.com/search/commits?q=author-email:%s", email)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+gh.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	if result.TotalCount > 0 && len(result.Items) > 0 {
		login := result.Items[0].Author.Login
		if login != "" {
			helpers.Log.Info().Msgf("Resolved email %s → @%s (via commit search)", email, login)
			return login
		}
	}

	return ""
}

// ResolveAuthor takes an author name and email and returns the best display name
func (gh *GitHubService) ResolveAuthor(name, email string) string {
	if username := gh.ResolveUsername(email); username != "" {
		return "@" + strings.ToLower(username)
	}
	return name
}
