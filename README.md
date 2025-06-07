# gh-reassign-reviewer

A GitHub CLI extension to re-request reviews from users who have already been requested on a pull request.

---

## Overview

**gh-reassign-reviewer** is a GitHub CLI extension that allows you to easily re-request reviews from users who have previously reviewed or commented on a pull request. This is useful for refreshing review requests to collaborators who have already participated in the PR discussion.

---

## Installation

You can install this extension using the GitHub CLI:

```sh
gh extension install ryo246912/gh-reassign-reviewer
```

Or clone this repository and build manually:

```sh
git clone https://github.com/ryo246912/gh-reassign-reviewer.git
cd gh-reassign-reviewer
go build -o gh-reassign-reviewer main.go
```

Move the binary to a directory in your `PATH`, or use it directly.

---

## Usage

Run the following command in your repository:

```sh
gh reassign-reviewer
```

Or specify a PR number:

```sh
gh reassign-reviewer <PR number>
```

- Select a reviewer from the list and confirm.
- The tool will re-request a review from the selected user.

---

## Configuration

No special configuration is required.
Make sure you are authenticated with the GitHub CLI (`gh auth login`).

---

## License

MIT License
