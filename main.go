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

// 共通のPR情報型
type PullRequestInfo struct {
	Number    int
	Title     string
	User      string
	State     string
	Draft     bool
	UpdatedAt string
	CreatedAt string
}

// 自分がアサインされているPRの一覧を取得し、選択肢を表示
func getPRNumber(client *api.RESTClient, gqlClient api.GraphQLClient, owner, repo, self string) (int, error) {
	prs, err := getAssignedPullRequests(gqlClient, owner, repo, self)
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
		items[i] = fmt.Sprintf("#%-6d %-50s %-15s %-20s %-20s", pr.Number, title, pr.User, state, pr.UpdatedAt)
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
func getAssignedPullRequests(client api.GraphQLClient, owner, repo, self string) ([]PullRequestInfo, error) {
	query := `
query ($query: String!, $first: Int = 100, $endCursor: String) {
  search(
    type: ISSUE,
    query: $query,
    first: $first,
    after: $endCursor
  ) {
    nodes {
      ... on PullRequest {
        number
        title
        state
        isDraft
        updatedAt
        createdAt
        author {
          login
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

	variables := map[string]interface{}{
		"query":     fmt.Sprintf("repo:%s/%s is:pr state:open assignee:@me sort:created-desc", owner, repo),
		"endCursor": nil,
	}

	var resp struct {
		Search struct {
			Nodes []struct {
				Number    int    `json:"number"`
				Title     string `json:"title"`
				State     string `json:"state"`
				IsDraft   bool   `json:"isDraft"`
				UpdatedAt string `json:"updatedAt"`
				CreatedAt string `json:"createdAt"`
				Author    struct {
					Login string `json:"login"`
				} `json:"author"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"search"`
	}

	err := client.Do(query, variables, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests (GraphQL): %w", err)
	}
	assigned := make([]PullRequestInfo, 0, len(resp.Search.Nodes))
	for _, pr := range resp.Search.Nodes {
		assigned = append(assigned, PullRequestInfo{
			Number:    pr.Number,
			Title:     pr.Title,
			User:      pr.Author.Login,
			State:     pr.State,
			Draft:     pr.IsDraft,
			UpdatedAt: pr.UpdatedAt,
			CreatedAt: pr.CreatedAt,
		})
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

	gqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return fmt.Errorf("failed to create GraphQL client: %w", err)
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
		prNumber, err = getPRNumber(client, *gqlClient, repo.Owner, repo.Name, self)
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
	for {
		fmt.Printf("You selected: %s. Is this correct? (y/n): ", selectedReviewer)
		if _, err := fmt.Scan(&confirm); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		confirmLower := strings.ToLower(confirm)
		if confirmLower == "yes" || confirmLower == "y" {
			break
		} else if confirmLower == "no" || confirmLower == "n" {
			fmt.Println("Reviewer selection cancelled")
			return nil
		} else {
			fmt.Println("Please enter 'y' or 'n'.")
		}
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
