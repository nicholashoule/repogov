package repogov

import (
	"os"
	"path/filepath"
)

// DetectPlatforms inspects the repository at root and returns the names of
// detected repository platforms based on the presence of well-known
// configuration files and directories:
//
//   - "github"    — .github/ directory exists
//   - "gitlab"    — .gitlab/ directory or .gitlab-ci.* file exists
//   - "bitbucket" — bitbucket-pipelines.* file exists
//
// The returned slice is in stable alphabetical order. An empty slice means
// no platform markers were found.
func DetectPlatforms(root string) []string {
	var platforms []string

	// Bitbucket: bitbucket-pipelines.yml (or any bitbucket-pipelines.* variant)
	if matches, _ := filepath.Glob(filepath.Join(root, "bitbucket-pipelines.*")); len(matches) > 0 {
		platforms = append(platforms, "bitbucket")
	}

	// GitHub: .github/ directory
	if info, err := os.Stat(filepath.Join(root, ".github")); err == nil && info.IsDir() {
		platforms = append(platforms, "github")
	}

	// GitLab: .gitlab/ directory or .gitlab-ci.* file
	gitlabDetected := false
	if info, err := os.Stat(filepath.Join(root, ".gitlab")); err == nil && info.IsDir() {
		gitlabDetected = true
	}
	if !gitlabDetected {
		if matches, _ := filepath.Glob(filepath.Join(root, ".gitlab-ci.*")); len(matches) > 0 {
			gitlabDetected = true
		}
	}
	if gitlabDetected {
		platforms = append(platforms, "gitlab")
	}

	return platforms
}
