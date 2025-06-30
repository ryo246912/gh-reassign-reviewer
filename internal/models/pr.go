package models

// PullRequestInfo represents PR metadata
type PullRequestInfo struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	User      string `json:"user"`
	State     string `json:"state"`
	Draft     bool   `json:"draft"`
	UpdatedAt string `json:"updated_at"`
	CreatedAt string `json:"created_at"`
}

// User represents a GitHub user
type User struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// Review represents a PR review
type Review struct {
	User User `json:"user"`
}

// Comment represents a PR comment
type Comment struct {
	User User `json:"user"`
}
