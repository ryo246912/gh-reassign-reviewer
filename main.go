package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// 自分がアサインされているPRの一覧を取得し、選択肢を表示
func getPRNumber(client *api.RESTClient, owner, repo, self string) (int, error) {
	prs, err := getAssignedPullRequests(client, owner, repo, self)
	if err != nil {
		return 0, err
	}
	if len(prs) == 0 {
		return 0, fmt.Errorf("No assigned pull requests found.")
	}

	items := make([]string, len(prs))
	for i, pr := range prs {
		state := pr.State
		if pr.Draft {
			state += " (Draft)"
		}
		title := pr.Title
		if len(title) > 75 {
			title = title[:75] + "..."
		}
		items[i] = fmt.Sprintf("#%-6d %-50s %-15s %-20s %-20s", pr.Number, title, pr.User.Login, state, pr.UpdatedAt)
	}
	prompt := promptui.Select{
		Label: "Select PR",
		Items: items,
		Size:  12,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return 0, fmt.Errorf("prompt failed: %w", err)
	}
	return prs[idx].Number, nil
}

// PRのレビュー履歴・コメント履歴から自分以外のユーザー（bot除外）を抽出
func getReviewersAndCommenters(client *api.RESTClient, owner, repo string, prNumber int, self string) ([]string, error) {
	userSet := make(map[string]struct{})

	// reviews
	reviewPath := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", owner, repo, prNumber)
	var reviews []struct {
		User struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"user"`
	}
	if err := client.Get(reviewPath, &reviews); err != nil {
		return nil, fmt.Errorf("failed to fetch reviews: %w", err)
	}
	for _, review := range reviews {
		login := review.User.Login
		typeStr := review.User.Type
		if login != self && !strings.HasSuffix(login, "[bot]") && typeStr != "Bot" && login != "" {
			userSet[login] = struct{}{}
		}
	}

	// issue comments
	commentPath := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, prNumber)
	var comments []struct {
		User struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"user"`
	}
	if err := client.Get(commentPath, &comments); err != nil {
		return nil, fmt.Errorf("failed to fetch comments: %w", err)
	}
	for _, comment := range comments {
		login := comment.User.Login
		typeStr := comment.User.Type
		if login != self && !strings.HasSuffix(login, "[bot]") && typeStr != "Bot" && login != "" {
			userSet[login] = struct{}{}
		}
	}

	var users []string
	for u := range userSet {
		users = append(users, u)
	}
	return users, nil
}

func reassignReviewers(client *api.RESTClient, owner, repo string, prNumber int, reviewers []string) error {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/requested_reviewers", owner, repo, prNumber)

	jsonBody, err := json.Marshal(map[string]interface{}{
		"reviewers": reviewers,
	})
	if err != nil {
		return fmt.Errorf("failed to encode request body: %w", err)
	}

	var response interface{}
	err = client.Post(path, bytes.NewReader(jsonBody), &response)
	if err != nil {
		return fmt.Errorf("failed to assign reviewers: %w", err)
	}
	return nil
}

// 自分のGitHubユーザー名をAPIで取得
func getCurrentUserLogin(client *api.RESTClient) (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := client.Get("user", &user); err != nil {
		return "", fmt.Errorf("failed to fetch current user: %w", err)
	}
	return user.Login, nil
}

// 自分がassigneeのPR一覧を取得
func getAssignedPullRequests(client *api.RESTClient, owner, repo, self string) ([]struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	State     string `json:"state"`
	Draft     bool   `json:"draft"`
	UpdatedAt string `json:"updated_at"`
	CreatedAt string `json:"created_at"`
}, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls?state=open&per_page=100", owner, repo)
	var pulls []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		State     string `json:"state"`
		Draft     bool   `json:"draft"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
	}
	if err := client.Get(path, &pulls); err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}
	var assigned []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		State     string `json:"state"`
		Draft     bool   `json:"draft"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	}
	for _, pr := range pulls {
		for _, a := range pr.Assignees {
			if a.Login == self {
				assigned = append(assigned, struct {
					Number int    `json:"number"`
					Title  string `json:"title"`
					User   struct {
						Login string `json:"login"`
					} `json:"user"`
					State     string `json:"state"`
					Draft     bool   `json:"draft"`
					UpdatedAt string `json:"updated_at"`
					CreatedAt string `json:"created_at"`
				}{
					Number: pr.Number,
					Title:  pr.Title,
					User: struct {
						Login string `json:"login"`
					}{Login: pr.User.Login},
					State:     pr.State,
					Draft:     pr.Draft,
					UpdatedAt: pr.UpdatedAt,
					CreatedAt: pr.CreatedAt,
				})
				break
			}
		}
	}
	return assigned, nil
}

func runCommand() error {
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	self, err := getCurrentUserLogin(client)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	var prNumber int
	if len(os.Args) >= 2 {
		prNumber, err = strconv.Atoi(os.Args[1])
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}
	} else {
		prNumber, err = getPRNumber(client, repo.Owner, repo.Name, self)
		if err != nil {
			return fmt.Errorf("failed to get PR number: %w", err)
		}
	}

	reviewers, err := getReviewersAndCommenters(client, repo.Owner, repo.Name, prNumber, self)
	if err != nil {
		return fmt.Errorf("failed to get reviewers and commenters: %w", err)
	}

	if len(reviewers) == 0 {
		fmt.Println("No available reviewers to re-request.")
		return nil
	}

	fmt.Println("Available reviewers:")
	for i, reviewer := range reviewers {
		fmt.Printf("%d. %s\n", i+1, reviewer)
	}

	// reviewerをpromptuiで選択
	prompt := promptui.Select{
		Label: "Select reviewer",
		Items: reviewers,
		Size:  12,
	}
	_, selectedReviewer, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("reviewer selection failed: %w", err)
	}

	var confirm string
	fmt.Printf("You selected: %s. Is this correct? (y/n): ", selectedReviewer)
	if _, err := fmt.Scan(&confirm); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if strings.ToLower(confirm) != "yes" && strings.ToLower(confirm) != "y" {
		fmt.Println("Reviewer selection cancelled")
		return nil
	}

	err = reassignReviewers(client, repo.Owner, repo.Name, prNumber, []string{selectedReviewer})
	if err != nil {
		return fmt.Errorf("failed to reassign reviewers: %w", err)
	}

	fmt.Printf("Successfully reassigned reviewer: %s\n", selectedReviewer)
	return nil
}

func main() {
	cmd := &cobra.Command{
		Use:   "reassign-reviewer",
		Short: "Reassign reviewers who have already been requested",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand()
		},
	}
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
