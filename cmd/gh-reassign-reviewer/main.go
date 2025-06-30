package main

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/ryo246912/gh-reassign-reviewer/internal/github"
	"github.com/ryo246912/gh-reassign-reviewer/internal/service"
	"github.com/ryo246912/gh-reassign-reviewer/internal/ui"
	"github.com/spf13/cobra"
)

// RepositoryAdapter adapts repository.Repository to our interface
type RepositoryAdapter struct {
	repo *repository.Repository
}

func (r *RepositoryAdapter) GetOwner() string {
	return r.repo.Owner
}

func (r *RepositoryAdapter) GetName() string {
	return r.repo.Name
}

func runCommand() error {
	// Get current repository
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Initialize GitHub client
	client, err := github.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Create service with dependency injection
	repoAdapter := &RepositoryAdapter{repo: &repo}
	prompter := &ui.DefaultPrompter{}
	reassignService := service.NewReassignService(client, repoAdapter, prompter)

	// Process the reassignment
	err = reassignService.ProcessReassignment(os.Args)
	if err != nil {
		return err
	}

	fmt.Println("Successfully reassigned reviewer")
	return nil
}

func main() {
	cmd := &cobra.Command{
		Use:   "reassign-reviewer",
		Short: "Reassign reviewers who have already been requested",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand()
		},
		SilenceUsage: true,
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
