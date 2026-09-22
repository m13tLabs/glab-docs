package util

import (
	"net/url"
	"os/exec"
	"strings"
)

func FindGitRepositoryRoot() (string, error) {
	path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(path)), nil
}

// FindGitProjectPath returns the "group/subgroup/project" path (no host, no scheme, no `.git`
// suffix) of the local repository's "origin" remote, for resolving a component's own address in
// the generated usage snippet when neither --component-prefix nor $CI_PROJECT_PATH is available
// (i.e. glab-docs run locally outside a GitLab CI job).
func FindGitProjectPath() (string, error) {
	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", err
	}

	return parseGitProjectPath(strings.TrimSpace(string(remote))), nil
}

// parseGitProjectPath extracts the project path from a git remote URL, in either its scp-like
// form (git@host:group/project.git) or a URL form (https://host/group/project.git,
// ssh://git@host:port/group/project.git).
func parseGitProjectPath(remoteURL string) string {
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	if !strings.Contains(remoteURL, "://") {
		if _, path, found := strings.Cut(remoteURL, ":"); found {
			return strings.TrimPrefix(path, "/")
		}
		return ""
	}

	parsed, err := url.Parse(remoteURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Path, "/")
}
