package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type GitHubCommitResponse struct {
	SHA string `json:"sha"`
}

// resolveGitHubRef resolves a GitHub reference (tag, branch, or commit) to a full commit SHA
func resolveGitHubRef(owner, repo, ref string) (string, error) {
	// If no ref specified, use HEAD (default branch)
	if ref == "" {
		ref = "HEAD"
	}

	// GitHub API endpoint to get commit info
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, ref)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header (GitHub API requires it)
	req.Header.Set("User-Agent", "go-npm")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch GitHub ref %s/%s#%s: %w", owner, repo, ref, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error for %s/%s#%s: %d %s", owner, repo, ref, resp.StatusCode, string(body))
	}

	var commitResp GitHubCommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&commitResp); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	if commitResp.SHA == "" {
		return "", fmt.Errorf("no commit SHA in response for %s/%s#%s", owner, repo, ref)
	}

	return commitResp.SHA, nil
}

// buildGitHubTarballURL constructs the GitHub tarball download URL for a commit SHA
func buildGitHubTarballURL(owner, repo, commitSHA string) string {
	return fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz", owner, repo, commitSHA)
}

// buildGitHubResolvedURL constructs the lock file resolved URL in npm format
func buildGitHubResolvedURL(owner, repo, commitSHA string) string {
	return fmt.Sprintf("git+ssh://git@github.com/%s/%s.git#%s", owner, repo, commitSHA)
}

// convertGitURLToTarball converts git URLs to GitHub tarball URLs
// Handles formats like:
// - git+ssh://git@github.com/owner/repo.git#commit
// - git+https://github.com/owner/repo.git#commit
// - git://github.com/owner/repo.git#commit
func convertGitURLToTarball(gitURL string) (tarballURL string, filename string, isGitURL bool) {
	// Pattern to match GitHub git URLs
	// Matches: git+ssh://git@github.com/owner/repo.git#commit
	//          git+https://github.com/owner/repo.git#commit
	//          git://github.com/owner/repo.git#commit
	gitPattern := regexp.MustCompile(`^git\+?(?:ssh|https)?://(?:git@)?github\.com[:/]([^/]+)/([^#]+?)(?:\.git)?#(.+)$`)

	matches := gitPattern.FindStringSubmatch(gitURL)
	if len(matches) != 4 {
		return "", "", false
	}

	owner := matches[1]
	repo := strings.TrimSuffix(matches[2], ".git")
	commitSHA := matches[3]

	tarballURL = buildGitHubTarballURL(owner, repo, commitSHA)
	filename = fmt.Sprintf("%s.tar.gz", commitSHA)

	return tarballURL, filename, true
}
