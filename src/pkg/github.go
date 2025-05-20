package pkg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// PullRequest struct contains the input parameters for the command
type PullRequest struct {
	Token  string
	Client *github.Client
	Info   struct {
		Owner       string
		Repo        string
		Number      int
		Title       string
		Description string
		Requirement string
	}
	PR      *github.PullRequest
	Context context.Context
}

// Body struct contains the input parameters for the command
type Body struct {
	Issue struct {
		Number      int      `json:"number"`
		PullRequest struct{} `json:"pull_request"`
	} `json:"issue"`
	Repository struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repository"`
	Comment struct {
		Body string `json:"body"`
	} `json:"comment"`
}

// GetPR returns a PullRequest struct
func GetPR(ctx context.Context) (*PullRequest, error) {
	// Get environment variables
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("required GITHUB_TOKEN environment variable is not set")
	}

	// Get event file path
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, fmt.Errorf("required GITHUB_EVENT_PATH environment variable is not set")
	}

	// Read event file
	eventFile, err := os.ReadFile(eventPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read event file: %v", err)
	}

	// Parse event file
	var body Body
	if err := json.Unmarshal(eventFile, &body); err != nil {
		return nil, fmt.Errorf("failed to parse event file: %v", err)
	}

	// Extract PR number, owner, and repository name
	prNumber := body.Issue.Number
	owner := body.Repository.Owner.Login
	repo := body.Repository.Name

	// Validate PR number
	if prNumber == 0 {
		return nil, fmt.Errorf("invalid PR number")
	}

	// Initialize GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Get Pull Request
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	// Return PullRequest struct
	var title, description string = "", ""
	if pr.Title != nil {
		title = *pr.Title
	}
	if pr.Body != nil {
		description = *pr.Body
	}
	return &PullRequest{
		Token:  githubToken,
		Client: client,
		Info: struct {
			Owner       string
			Repo        string
			Number      int
			Title       string
			Description string
			Requirement string
		}{
			Owner:       owner,
			Repo:        repo,
			Number:      prNumber,
			Title:       title,
			Description: description,
			Requirement: body.Comment.Body,
		},
		PR:      pr,
		Context: ctx,
	}, nil
}

// File represents a file in the diff
type File struct {
	Path       string
	OldPath    string
	IsNew      bool
	IsDeleted  bool
	IsRenamed  bool
	Hunks      []Hunk
	IsBinary   bool
	BinaryDiff string
}

// String returns the diff of the file in GitHub Flavored Markdown format
func (f *File) String() string {
	hunks := "# File Info\n"
	hunks += fmt.Sprintf("Path: %s\n", f.Path)
	hunks += fmt.Sprintf("OldPath: %s\n", f.OldPath)
	hunks += fmt.Sprintf("IsNew: %t\n", f.IsNew)
	hunks += fmt.Sprintf("IsDeleted: %t\n", f.IsDeleted)
	hunks += fmt.Sprintf("IsRenamed: %t\n", f.IsRenamed)
	hunks += fmt.Sprintf("IsBinary: %t\n", f.IsBinary)
	if f.IsBinary {
		hunks += fmt.Sprintf("BinaryDiff: %s\n", f.BinaryDiff)
	}
	hunks += "\n# Hunks\n"
	cnt := 0
	for _, hunk := range f.Hunks {
		hunks += hunk.Header + "\n"
		for _, line := range hunk.Lines {
			cnt++
			hunks += fmt.Sprintf("%d: %s\n", cnt, line)
		}
	}
	return hunks
}

// Hunk represents a hunk in the diff
type Hunk struct {
	Header string
	Lines  []string
}

// parseDiff parses the diff of the pull request
func parseDiff(r io.Reader) ([]*File, error) {
	var files []*File
	var currentFile *File
	var currentHunk *Hunk

	// Parse diff
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "diff --git a/"):
			// New file
			if currentFile != nil {
				if currentHunk != nil && len(currentHunk.Lines) > 0 {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				files = append(files, currentFile)
				currentHunk = nil
			}
			currentFile = &File{Hunks: []Hunk{}}
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentFile.OldPath = strings.TrimPrefix(parts[2], "a/")
				currentFile.Path = strings.TrimPrefix(parts[3], "b/")
			}
		case currentFile == nil:
			// Skip lines before the first file
			continue
		case strings.HasPrefix(line, "new file mode"):
			// New file
			currentFile.IsNew = true
		case strings.HasPrefix(line, "deleted file mode"):
			// Deleted file
			currentFile.IsDeleted = true
		case strings.HasPrefix(line, "rename from "):
			// Renamed file
			currentFile.IsRenamed = true
			currentFile.OldPath = strings.TrimPrefix(line, "rename from ")
		case strings.HasPrefix(line, "rename to "):
			// Renamed file
			currentFile.IsRenamed = true
			currentFile.Path = strings.TrimPrefix(line, "rename to ")
		case strings.HasPrefix(line, "--- a/"):
			// Old file
			if currentFile.OldPath == "" {
				currentFile.OldPath = strings.TrimPrefix(line, "--- a/")
			}
			if strings.TrimPrefix(line, "--- a/") == "/dev/null" {
				currentFile.IsNew = true
			}
		case strings.HasPrefix(line, "+++ b/"):
			// New file
			if currentFile.Path == "" {
				currentFile.Path = strings.TrimPrefix(line, "+++ b/")
			}
			if strings.TrimPrefix(line, "+++ b/") == "/dev/null" {
				currentFile.IsDeleted = true
			}
		case strings.HasPrefix(line, "Binary files "):
			// Binary file
			currentFile.IsBinary = true
			currentFile.BinaryDiff = line
			files = append(files, currentFile)
			currentFile = nil
			currentHunk = nil
		case strings.HasPrefix(line, "@@") && !currentFile.IsBinary:
			// Hunk
			if currentHunk != nil && len(currentHunk.Lines) > 0 {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}
			currentHunk = &Hunk{Header: line, Lines: []string{}}
		case currentHunk != nil && !currentFile.IsBinary:
			// Hunk line
			currentHunk.Lines = append(currentHunk.Lines, line)
		}
	}

	// Add last file
	if currentFile != nil {
		if currentHunk != nil && len(currentHunk.Lines) > 0 {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		if currentFile.Path != "" || currentFile.IsBinary {
			files = append(files, currentFile)
		}
	}

	// Return files
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// GetDiffFiles returns the diff of the pull request as a string
func (pr *PullRequest) GetDiffFiles() ([]*File, error) {
	// Get the diff from the GitHub API
	api_url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d.diff", pr.Info.Owner, pr.Info.Repo, pr.Info.Number)
	req, err := http.NewRequest("GET", api_url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", pr.Token))
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get diff: status %d", resp.StatusCode)
	}

	diffBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read diff body: %v", err)
	}

	// Parse diff
	return parseDiff(bytes.NewReader(diffBytes))
}

// Message struct contains the input parameters for the command
type Message struct {
	Body     string
	Path     string
	Position int
}

// CreatePRComments creates a comment on a pull request
func (pr *PullRequest) CreatePRComments(msgs []*Message) error {
	for _, m := range msgs {
		comment := &github.PullRequestComment{
			Body:     github.String(m.Body),
			Path:     github.String(m.Path),
			CommitID: pr.PR.Head.SHA,
			Position: github.Int(m.Position),
		}
		if _, _, err := pr.Client.PullRequests.CreateComment(pr.Context, pr.Info.Owner, pr.Info.Repo, pr.Info.Number, comment); err != nil {
			return err
		}
	}
	fmt.Println("✅ Successfully created comments on PR #" + fmt.Sprintf("%d", pr.Info.Number))
	return nil
}

// CreateIssueComment create a comment on an issue
func (pr *PullRequest) CreateIssueComment(msg string) error {
	comment := &github.IssueComment{
		Body: github.String(msg),
	}
	if _, _, err := pr.Client.Issues.CreateComment(pr.Context, pr.Info.Owner, pr.Info.Repo, pr.Info.Number, comment); err != nil {
		return err
	}
	fmt.Println("✅ Successfully created comment on issue #" + fmt.Sprintf("%d", pr.Info.Number))
	return nil
}

// GetReadmeContent returns the content of the README.md file
func (pr *PullRequest) GetReadmeContent() (string, error) {
	readme, resp, err := pr.Client.Repositories.GetReadme(pr.Context, pr.Info.Owner, pr.Info.Repo, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}

	content, err := readme.GetContent()
	if err != nil {
		return "", err
	}

	return content, nil
}
