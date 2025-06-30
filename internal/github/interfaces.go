package github

import (
	"github.com/ryo246912/gh-reassign-reviewer/internal/models"
)

// GitHubClient defines the interface for GitHub operations
type GitHubClient interface {
	GetCurrentUserLogin() (string, error)
	GetAssignedPRs(owner, repo, self string) ([]models.PullRequestInfo, error)
	GetReviewersAndCommenters(owner, repo string, prNumber int, self string) ([]string, error)
	ReassignReviewers(owner, repo string, prNumber int, reviewers []string) error
}

// RepositoryInfo defines repository information interface
type RepositoryInfo interface {
	GetOwner() string
	GetName() string
}

// Ensure Client implements GitHubClient interface
var _ GitHubClient = (*Client)(nil)
