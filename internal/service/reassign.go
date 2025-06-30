package service

import (
	"fmt"
	"strconv"

	"github.com/ryo246912/gh-reassign-reviewer/internal/github"
	"github.com/ryo246912/gh-reassign-reviewer/internal/ui"
)

// ReassignService contains the business logic
type ReassignService struct {
	client   github.GitHubClient
	repo     github.RepositoryInfo
	prompter ui.Prompter
}

// NewReassignService creates a new service instance
func NewReassignService(client github.GitHubClient, repo github.RepositoryInfo, prompter ui.Prompter) *ReassignService {
	return &ReassignService{
		client:   client,
		repo:     repo,
		prompter: prompter,
	}
}

// ProcessReassignment handles the complete workflow
func (s *ReassignService) ProcessReassignment(args []string) error {
	// Get current user
	self, err := s.client.GetCurrentUserLogin()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Get PR number from args or prompt
	prNumber, err := s.getPRNumber(args, self)
	if err != nil {
		return fmt.Errorf("failed to get PR number: %w", err)
	}

	// Get available reviewers
	reviewers, err := s.client.GetReviewersAndCommenters(s.repo.GetOwner(), s.repo.GetName(), prNumber, self)
	if err != nil {
		return fmt.Errorf("failed to get reviewers and commenters: %w", err)
	}

	if len(reviewers) == 0 {
		return fmt.Errorf("no available reviewers to re-request")
	}

	// Select reviewer
	selectedReviewer, err := s.prompter.SelectReviewer(reviewers)
	if err != nil {
		return fmt.Errorf("failed to select reviewer: %w", err)
	}

	// Confirm selection
	confirmed, err := s.prompter.ConfirmSelection(selectedReviewer)
	if err != nil {
		return fmt.Errorf("failed to confirm selection: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("reviewer selection cancelled")
	}

	// Reassign reviewer
	err = s.client.ReassignReviewers(s.repo.GetOwner(), s.repo.GetName(), prNumber, []string{selectedReviewer})
	if err != nil {
		return fmt.Errorf("failed to reassign reviewers: %w", err)
	}

	return nil
}

// getPRNumber gets PR number from args or prompts user
func (s *ReassignService) getPRNumber(args []string, self string) (int, error) {
	if len(args) >= 2 {
		prNumber, err := strconv.Atoi(args[1])
		if err != nil {
			return 0, fmt.Errorf("invalid PR number: %w", err)
		}
		if prNumber <= 0 {
			return 0, fmt.Errorf("PR number must be positive")
		}
		return prNumber, nil
	}

	// No PR number provided, prompt user
	prs, err := s.client.GetAssignedPRs(s.repo.GetOwner(), s.repo.GetName(), self)
	if err != nil {
		return 0, fmt.Errorf("failed to get assigned PRs: %w", err)
	}

	return s.prompter.SelectPR(prs)
}

// ValidateReviewers checks if reviewers list is valid
func (s *ReassignService) ValidateReviewers(reviewers []string, self string) error {
	if len(reviewers) == 0 {
		return fmt.Errorf("no reviewers provided")
	}

	for _, reviewer := range reviewers {
		if reviewer == "" {
			return fmt.Errorf("reviewer name cannot be empty")
		}
		if reviewer == self {
			return fmt.Errorf("cannot assign yourself as reviewer")
		}
	}

	return nil
}

// GetAvailableReviewers returns filtered list of available reviewers
func (s *ReassignService) GetAvailableReviewers(prNumber int, self string) ([]string, error) {
	reviewers, err := s.client.GetReviewersAndCommenters(s.repo.GetOwner(), s.repo.GetName(), prNumber, self)
	if err != nil {
		return nil, err
	}

	// Additional validation can be added here
	validReviewers := make([]string, 0, len(reviewers))
	for _, reviewer := range reviewers {
		if reviewer != self && reviewer != "" {
			validReviewers = append(validReviewers, reviewer)
		}
	}

	return validReviewers, nil
}
