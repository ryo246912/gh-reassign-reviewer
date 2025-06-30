package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/ryo246912/gh-reassign-reviewer/internal/models"
)

// Client wraps GitHub API clients
type Client struct {
	rest api.RESTClient
	gql  api.GraphQLClient
}

func NewClient() (*Client, error) {
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	gqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}

	return &Client{
		rest: *restClient,
		gql:  *gqlClient,
	}, nil
}

// GetCurrentUserLogin fetches current user's login
func (c *Client) GetCurrentUserLogin() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := c.rest.Get("user", &user); err != nil {
		return "", fmt.Errorf("failed to fetch current user: %w", err)
	}
	return user.Login, nil
}

// GetAssignedPRs fetches assigned pull requests using GraphQL
func (c *Client) GetAssignedPRs(owner, repo, self string) ([]models.PullRequestInfo, error) {
	// NOTE: https://github.com/cli/go-gh/blob/a08820a13f257d6c5b4cb86d37db559ec6d14577/example_gh_test.go#L233
	// query := `
	// 	query ($query: String!, $first: Int = 100, $endCursor: String) {
	// 		search(
	// 			type: ISSUE,
	// 			query: $query,
	// 			first: $first,
	// 			after: $endCursor
	// 		) {
	// 			nodes {
	// 				... on PullRequest {
	// 					number
	// 					title
	// 					state
	// 					isDraft
	// 					updatedAt
	// 					createdAt
	// 					author {
	// 						login
	// 					}
	// 				}
	// 			}
	// 			pageInfo {
	// 				hasNextPage
	// 				endCursor
	// 			}
	// 		}
	// 	}
	// `
	var q struct {
		Search struct {
			Nodes []struct {
				PullRequest struct {
					Number    int
					Title     string
					State     string
					IsDraft   bool
					UpdatedAt string
					CreatedAt string
					Author    struct {
						Login string
					}
				} `graphql:"... on PullRequest"`
			}
			PageInfo struct {
				HasNextPage bool
				EndCursor   string
			}
		} `graphql:"search(type: ISSUE, query: $query, first: $first, after: $endCursor)"`
	}

	variables := map[string]interface{}{
		"query":     graphql.String(fmt.Sprintf("repo:%s/%s is:pr state:open assignee:%s sort:created-desc", owner, repo, self)),
		"first":     graphql.Int(100),
		"endCursor": (*graphql.String)(nil),
	}

	err := c.gql.Query("", &q, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	assigned := make([]models.PullRequestInfo, 0, len(q.Search.Nodes))
	for _, node := range q.Search.Nodes {
		pr := node.PullRequest
		assigned = append(assigned, models.PullRequestInfo{
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

// GetReviewersAndCommenters extracts users from PR reviews and comments
func (c *Client) GetReviewersAndCommenters(owner, repo string, prNumber int, self string) ([]string, error) {
	userSet := make(map[string]struct{}) // Use map as set

	// Get reviews
	reviewPath := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", owner, repo, prNumber)
	var reviews []models.Review
	if err := c.rest.Get(reviewPath, &reviews); err != nil {
		return nil, fmt.Errorf("failed to fetch reviews: %w", err)
	}

	for _, review := range reviews {
		login := review.User.Login
		typeStr := review.User.Type
		if c.isValidUser(login, typeStr, self) {
			userSet[login] = struct{}{}
		}
	}

	// Get issue comments
	commentPath := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, prNumber)
	var comments []models.Comment
	if err := c.rest.Get(commentPath, &comments); err != nil {
		return nil, fmt.Errorf("failed to fetch comments: %w", err)
	}

	for _, comment := range comments {
		login := comment.User.Login
		typeStr := comment.User.Type
		if c.isValidUser(login, typeStr, self) {
			userSet[login] = struct{}{}
		}
	}

	// Convert map to slice
	users := make([]string, 0, len(userSet))
	for u := range userSet {
		users = append(users, u)
	}
	return users, nil
}

// isValidUser checks if user should be included as potential reviewer
func (c *Client) isValidUser(login, userType, self string) bool {
	return login != self &&
		!strings.HasSuffix(login, "[bot]") &&
		userType != "Bot" &&
		login != ""
}

// ReassignReviewers sends review request to specified reviewers
func (c *Client) ReassignReviewers(owner, repo string, prNumber int, reviewers []string) error {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/requested_reviewers", owner, repo, prNumber)

	jsonBody, err := json.Marshal(map[string]interface{}{
		"reviewers": reviewers,
	})
	if err != nil {
		return fmt.Errorf("failed to encode request body: %w", err)
	}

	var response interface{}
	err = c.rest.Post(path, bytes.NewReader(jsonBody), &response)
	if err != nil {
		return fmt.Errorf("failed to assign reviewers: %w", err)
	}
	return nil
}
