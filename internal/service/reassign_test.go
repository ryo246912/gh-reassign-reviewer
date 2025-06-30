package service

import (
	"testing"

	"github.com/ryo246912/gh-reassign-reviewer/internal/github"
	"github.com/ryo246912/gh-reassign-reviewer/internal/models"
	"github.com/ryo246912/gh-reassign-reviewer/internal/ui"
)

// TestGetPRNumber tests PR number extraction from arguments
func TestReassignService_getPRNumber(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		mockPRs       []models.PullRequestInfo
		mockPRsError  error
		expectedPR    int
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid PR number from args",
			args:        []string{"program", "123"},
			expectedPR:  123,
			expectError: false,
		},
		{
			name:          "invalid PR number - not a number",
			args:          []string{"program", "abc"},
			expectError:   true,
			errorContains: "invalid PR number",
		},
		{
			name:          "invalid PR number - zero",
			args:          []string{"program", "0"},
			expectError:   true,
			errorContains: "PR number must be positive",
		},
		{
			name:          "invalid PR number - negative",
			args:          []string{"program", "-1"},
			expectError:   true,
			errorContains: "PR number must be positive",
		},
		{
			name: "no args - single PR available",
			args: []string{"program"},
			mockPRs: []models.PullRequestInfo{
				{Number: 456, Title: "Test PR", User: "testuser", State: "open"},
			},
			expectedPR:  456,
			expectError: false,
		},
		{
			name:          "no args - API error",
			args:          []string{"program"},
			mockPRsError:  github.NewAPIError("failed to fetch PRs"),
			expectError:   true,
			errorContains: "failed to get assigned PRs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			client := &github.MockClient{
				AssignedPRs:      tt.mockPRs,
				AssignedPRsError: tt.mockPRsError,
			}
			repo := &github.MockRepository{Owner: "owner", Name: "repo"}
			prompter := &ui.MockPrompter{
				SelectedPRNumber: tt.expectedPR,
			}
			service := NewReassignService(client, repo, prompter)

			// Test the method
			prNumber, err := service.getPRNumber(tt.args, "testuser")

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorContains != "" {
				if !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Error %q should contain %q", err.Error(), tt.errorContains)
				}
			}

			// Check result
			if !tt.expectError && prNumber != tt.expectedPR {
				t.Errorf("Expected PR number %d, got %d", tt.expectedPR, prNumber)
			}
		})
	}
}

// TestValidateReviewers tests reviewer validation logic
func TestReassignService_ValidateReviewers(t *testing.T) {
	tests := []struct {
		name          string
		reviewers     []string
		self          string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid reviewers",
			reviewers:   []string{"user1", "user2"},
			self:        "currentuser",
			expectError: false,
		},
		{
			name:          "empty reviewers list",
			reviewers:     []string{},
			self:          "currentuser",
			expectError:   true,
			errorContains: "no reviewers provided",
		},
		{
			name:          "empty reviewer name",
			reviewers:     []string{"user1", "", "user2"},
			self:          "currentuser",
			expectError:   true,
			errorContains: "reviewer name cannot be empty",
		},
		{
			name:          "self as reviewer",
			reviewers:     []string{"user1", "currentuser"},
			self:          "currentuser",
			expectError:   true,
			errorContains: "cannot assign yourself as reviewer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &github.MockClient{}
			repo := &github.MockRepository{Owner: "owner", Name: "repo"}
			prompter := &ui.MockPrompter{}
			service := NewReassignService(client, repo, prompter)

			err := service.ValidateReviewers(tt.reviewers, tt.self)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorContains != "" {
				if !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Error %q should contain %q", err.Error(), tt.errorContains)
				}
			}
		})
	}
}

// TestGetAvailableReviewers tests reviewer filtering logic
func TestReassignService_GetAvailableReviewers(t *testing.T) {
	tests := []struct {
		name              string
		prNumber          int
		self              string
		mockReviewers     []string
		mockError         error
		expectedCount     int
		expectedReviewers []string
		expectError       bool
	}{
		{
			name:              "valid reviewers excluding self",
			prNumber:          123,
			self:              "currentuser",
			mockReviewers:     []string{"user1", "user2", "currentuser", "user3"},
			expectedCount:     3,
			expectedReviewers: []string{"user1", "user2", "user3"},
			expectError:       false,
		},
		{
			name:        "API error",
			prNumber:    123,
			self:        "currentuser",
			mockError:   github.NewAPIError("failed"),
			expectError: true,
		},
		{
			name:              "empty reviewers with empty strings",
			prNumber:          123,
			self:              "currentuser",
			mockReviewers:     []string{"", "currentuser", ""},
			expectedCount:     0,
			expectedReviewers: []string{},
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &github.MockClient{
				ReviewersCommenters: tt.mockReviewers,
				ReviewersError:      tt.mockError,
			}
			repo := &github.MockRepository{Owner: "owner", Name: "repo"}
			prompter := &ui.MockPrompter{}
			service := NewReassignService(client, repo, prompter)

			reviewers, err := service.GetAvailableReviewers(tt.prNumber, tt.self)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if len(reviewers) != tt.expectedCount {
					t.Errorf("Expected %d reviewers, got %d", tt.expectedCount, len(reviewers))
				}

				// Check if expected reviewers are present
				for _, expected := range tt.expectedReviewers {
					found := false
					for _, actual := range reviewers {
						if actual == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected reviewer %q not found in result", expected)
					}
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
