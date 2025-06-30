package ui

import "github.com/ryo246912/gh-reassign-reviewer/internal/models"

// Prompter defines interface for user interaction
type Prompter interface {
	SelectPR(prs []models.PullRequestInfo) (int, error)
	SelectReviewer(reviewers []string) (string, error)
	ConfirmSelection(reviewer string) (bool, error)
}

// DefaultPrompter implements the actual prompting logic
type DefaultPrompter struct{}

// SelectPR prompts user to select a PR
func (p *DefaultPrompter) SelectPR(prs []models.PullRequestInfo) (int, error) {
	return SelectPR(prs)
}

// SelectReviewer prompts user to select a reviewer
func (p *DefaultPrompter) SelectReviewer(reviewers []string) (string, error) {
	return SelectReviewer(reviewers)
}

// ConfirmSelection prompts user to confirm selection
func (p *DefaultPrompter) ConfirmSelection(reviewer string) (bool, error) {
	return ConfirmSelection(reviewer)
}

// MockPrompter for testing
type MockPrompter struct {
	SelectedPRNumber int
	PRSelectionError error

	SelectedReviewer       string
	ReviewerSelectionError error

	ConfirmedSelection bool
	ConfirmationError  error

	// Call tracking
	SelectPRCalled         bool
	SelectReviewerCalled   bool
	ConfirmSelectionCalled bool
}

// SelectPR mocks PR selection
func (m *MockPrompter) SelectPR(prs []models.PullRequestInfo) (int, error) {
	m.SelectPRCalled = true
	return m.SelectedPRNumber, m.PRSelectionError
}

// SelectReviewer mocks reviewer selection
func (m *MockPrompter) SelectReviewer(reviewers []string) (string, error) {
	m.SelectReviewerCalled = true
	return m.SelectedReviewer, m.ReviewerSelectionError
}

// ConfirmSelection mocks confirmation
func (m *MockPrompter) ConfirmSelection(reviewer string) (bool, error) {
	m.ConfirmSelectionCalled = true
	return m.ConfirmedSelection, m.ConfirmationError
}
