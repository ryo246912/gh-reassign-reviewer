package github

import (
	"fmt"
	"strings"

	"github.com/ryo246912/gh-reassign-reviewer/internal/models"
)

// MockClient implements GitHubClient for testing
type MockClient struct {
	// Control test behavior
	CurrentUser         string
	CurrentUserError    error
	AssignedPRs         []models.PullRequestInfo
	AssignedPRsError    error
	ReviewersCommenters []string
	ReviewersError      error
	ReassignError       error

	// Track method calls
	GetCurrentUserLoginCalled       bool
	GetAssignedPRsCalled            bool
	GetReviewersAndCommentersCalled bool
	ReassignReviewersCalled         bool

	// Store call arguments for verification
	LastOwner     string
	LastRepo      string
	LastPRNumber  int
	LastReviewers []string
}

// GetCurrentUserLogin mocks the GitHub API call
func (m *MockClient) GetCurrentUserLogin() (string, error) {
	m.GetCurrentUserLoginCalled = true
	return m.CurrentUser, m.CurrentUserError
}

// GetAssignedPRs mocks the GraphQL API call
func (m *MockClient) GetAssignedPRs(owner, repo, self string) ([]models.PullRequestInfo, error) {
	m.GetAssignedPRsCalled = true
	m.LastOwner = owner
	m.LastRepo = repo
	return m.AssignedPRs, m.AssignedPRsError
}

// GetReviewersAndCommenters mocks the REST API calls
func (m *MockClient) GetReviewersAndCommenters(owner, repo string, prNumber int, self string) ([]string, error) {
	m.GetReviewersAndCommentersCalled = true
	m.LastOwner = owner
	m.LastRepo = repo
	m.LastPRNumber = prNumber
	return m.ReviewersCommenters, m.ReviewersError
}

// ReassignReviewers mocks the review request API call
func (m *MockClient) ReassignReviewers(owner, repo string, prNumber int, reviewers []string) error {
	m.ReassignReviewersCalled = true
	m.LastOwner = owner
	m.LastRepo = repo
	m.LastPRNumber = prNumber
	m.LastReviewers = reviewers
	return m.ReassignError
}

// Reset clears all tracking data for fresh test
func (m *MockClient) Reset() {
	m.GetCurrentUserLoginCalled = false
	m.GetAssignedPRsCalled = false
	m.GetReviewersAndCommentersCalled = false
	m.ReassignReviewersCalled = false
	m.LastOwner = ""
	m.LastRepo = ""
	m.LastPRNumber = 0
	m.LastReviewers = nil
}

// MockRepository implements repository information for testing
type MockRepository struct {
	Owner string
	Name  string
}

func (m *MockRepository) GetOwner() string {
	return m.Owner
}

func (m *MockRepository) GetName() string {
	return m.Name
}

// Helper functions for creating test data
func CreateTestPRs(count int) []models.PullRequestInfo {
	prs := make([]models.PullRequestInfo, count)
	for i := 0; i < count; i++ {
		prs[i] = models.PullRequestInfo{
			Number:    i + 1,
			Title:     fmt.Sprintf("Test PR #%d", i+1),
			User:      fmt.Sprintf("user%d", i+1),
			State:     "open",
			Draft:     i%2 == 0, // Alternate between draft and ready
			UpdatedAt: "2023-01-01T12:00:00Z",
			CreatedAt: "2023-01-01T10:00:00Z",
		}
	}
	return prs
}

func CreateTestReviewers(count int) []string {
	reviewers := make([]string, count)
	for i := 0; i < count; i++ {
		reviewers[i] = fmt.Sprintf("reviewer%d", i+1)
	}
	return reviewers
}

// Error helpers for testing error conditions
func NewUserNotFoundError() error {
	return fmt.Errorf("user not found")
}

func NewAPIError(message string) error {
	return fmt.Errorf("API error: %s", message)
}

func NewNetworkError() error {
	return fmt.Errorf("network connection failed")
}

// Validation helpers
func ValidateReviewerNames(reviewers []string) error {
	for _, reviewer := range reviewers {
		if reviewer == "" {
			return fmt.Errorf("reviewer name cannot be empty")
		}
		if strings.Contains(reviewer, "[bot]") {
			return fmt.Errorf("bot reviewers are not allowed: %s", reviewer)
		}
	}
	return nil
}
