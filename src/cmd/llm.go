package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	_ "embed"

	"github.com/lee-lou2/ai-code-reviewer/pkg"
)

//go:embed prompt/system.txt
var systemPrompt string

//go:embed prompt/user.txt
var userPrompt string

// Review struct contains the input parameters for the command
type Review struct {
	Body     string `json:"body"`
	Position int    `json:"position"`
}

// Reviews struct contains the input parameters for the command
type Reviews struct {
	Reviews []*Review `json:"reviews"`
	Summary string    `json:"summary"`
}

// GenReviews generates reviews for a file using Gemini or OpenAI
func GenReviews(ctx context.Context, pr *pkg.PullRequest, file *pkg.File, readmeContent string) (*Reviews, error) {
	// Generate content
	hunks := file.String()

	// Set system prompt
	language := os.Getenv("LANGUAGE")
	if language == "" {
		language = "English"
	}
	systemPrompt = fmt.Sprintf(systemPrompt, readmeContent)
	userPrompt = fmt.Sprintf(userPrompt, language, file.Path, pr.Info.Requirement, pr.Info.Title, pr.Info.Description, hunks)

	// Select LLM
	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	openaiModel := os.Getenv("OPENAI_MODEL")

	f := pkg.GenGemini
	if openaiApiKey != "" && openaiModel != "" {
		f = pkg.GenOpenAI
	}

	// Generate reviews
	result, err := f(ctx, userPrompt, systemPrompt)
	if err != nil {
		return nil, err
	}

	// Unmarshal the result
	var reviews Reviews
	if err := json.Unmarshal([]byte(result), &reviews); err != nil {
		return nil, err
	}
	return &reviews, nil
}
