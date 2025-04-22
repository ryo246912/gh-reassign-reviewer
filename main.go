package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
)

func getPRNumberFromArgs() (int, error) {
	if len(os.Args) < 2 {
		return 0, fmt.Errorf("PR number is required")
	}
	prNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return 0, fmt.Errorf("invalid PR number: %w", err)
	}
	return prNumber, nil
}

func fetchPRHTML(owner, repo string, prNumber int) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, prNumber)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PR page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func parseReviewersFromHTML(html string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var reviewers []string
	doc.Find(".assignee").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			// Extract the username from the href (e.g., /username)
			username := strings.TrimPrefix(href, "/")
			reviewers = append(reviewers, username)
		}
	})

	return reviewers, nil
}

func reassignReviewers(client *api.RESTClient, owner, repo string, prNumber int, reviewers []string) error {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/requested_reviewers", owner, repo, prNumber)

	// JSONにエンコード
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

func runCommand() error {
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	prNumber, err := getPRNumberFromArgs()
	if err != nil {
		return fmt.Errorf("failed to get PR number: %w", err)
	}

	html, err := fetchPRHTML(repo.Owner, repo.Name, prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR HTML: %w", err)
	}

	reviewers, err := parseReviewersFromHTML(html)
	if err != nil {
		return fmt.Errorf("failed to parse reviewers: %w", err)
	}

	if len(reviewers) == 0 {
		fmt.Println("No previously requested reviewers found")
		return nil
	}

	fmt.Println("Available reviewers:")
	for i, reviewer := range reviewers {
		fmt.Printf("%d. %s\n", i+1, reviewer)
	}

	// select a reviewer
	var selectedIndex int
	for {
		fmt.Print("Select a reviewer (enter number): ")
		_, err := fmt.Scan(&selectedIndex)
		if err != nil {
			fmt.Println("Invalid input, please try again")
			continue
		}
		if selectedIndex < 1 || selectedIndex > len(reviewers) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(reviewers))
			continue
		}
		break
	}

	selectedReviewer := reviewers[selectedIndex-1]

	// Confirm selection
	var confirm string
	fmt.Printf("You selected: %s. Is this correct? (y/n): ", selectedReviewer)
	fmt.Scan(&confirm)

	if strings.ToLower(confirm) != "yes" && strings.ToLower(confirm) != "y" {
		fmt.Println("Reviewer selection cancelled")
		return nil
	}

	err = reassignReviewers(client, repo.Owner, repo.Name, prNumber, []string{selectedReviewer})
	if err != nil {
		return fmt.Errorf("failed to reassign reviewers: %w", err)
	}

	fmt.Printf("Successfully reassigned %d reviewers\n", len(reviewers))
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

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
