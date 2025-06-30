package ui

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/ryo246912/gh-reassign-reviewer/internal/models"
)

func SelectPR(prs []models.PullRequestInfo) (int, error) {
	if len(prs) == 0 {
		return 0, fmt.Errorf("no assigned pull requests found")
	}

	items := make([]string, len(prs))
	for i, pr := range prs {
		state := pr.State
		if pr.Draft {
			state += " (Draft)"
		}
		title := pr.Title
		if len(title) > 75 {
			title = title[:72] + "..."
		}
		items[i] = fmt.Sprintf(
			"#%s %s %s %s %s",
			PadRight(fmt.Sprintf("%-6d", pr.Number), 7),
			PadRight(title, 75),
			PadRight(pr.User, 15),
			PadRight(state, 10),
			PadRight(pr.UpdatedAt, 20),
		)
	}

	prompt := promptui.Select{
		Label: "Select PR",
		Items: items,
		Size:  12,
		Searcher: func(input string, index int) bool {
			return strings.Contains(strings.ToLower(items[index]), input)
		},
		StartInSearchMode: true,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return 0, fmt.Errorf("prompt failed: %w", err)
	}
	return prs[idx].Number, nil
}

// SelectReviewer shows reviewer selection prompt
func SelectReviewer(reviewers []string) (string, error) {
	if len(reviewers) == 0 {
		return "", fmt.Errorf("no available reviewers")
	}

	fmt.Println("Available reviewers:")
	for i, reviewer := range reviewers {
		fmt.Printf("%d. %s\n", i+1, reviewer)
	}

	prompt := promptui.Select{
		Label: "Select reviewer",
		Items: reviewers,
		Size:  12,
		Searcher: func(input string, index int) bool {
			return strings.Contains(strings.ToLower(reviewers[index]), input)
		},
		StartInSearchMode: true,
	}

	_, selectedReviewer, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("reviewer selection failed: %w", err)
	}

	return selectedReviewer, nil
}

// ConfirmSelection asks for user confirmation
// Similar to Python's input() with validation
func ConfirmSelection(reviewer string) (bool, error) {
	var confirm string
	for {
		fmt.Printf("You selected: %s. Is this correct? (y/n): ", reviewer)
		if _, err := fmt.Scan(&confirm); err != nil {
			return false, fmt.Errorf("failed to read confirmation: %w", err)
		}
		confirmLower := strings.ToLower(confirm)
		switch confirmLower {
		case "yes", "y":
			return true, nil
		case "no", "n":
			return false, nil
		default:
			fmt.Println("Please enter 'y' or 'n'.")
		}
	}
}
