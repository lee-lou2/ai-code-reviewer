package pkg

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"google.golang.org/genai"
)

// GenGemini generates reviews for a file using Gemini
func GenGemini(ctx context.Context, userPrompt, systemPrompt string) (string, error) {
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiApiKey == "" || geminiModel == "" {
		return "", fmt.Errorf("GEMINI_API_KEY or GEMINI_MODEL is not set")
	}

	// Generate schema for the response
	schema := genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"reviews": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"body": {
							Type: genai.TypeString,
							Example: `🛠️ Refactor suggestion

"JSOResponse" appears to be a typo. You should use "JSONResponse" instead.

` + "```diff" + `
- return JSOResponse()
+ return JSONResponse()
` + "```",
							Description: `# Field Description
Review comment in GitHub Flavored Markdown format.

# Review Comment Format
🛠️ {Issue Type}

{Detailed review comment}

` + "```diff" + `
- {Before code}
+ {After code}
` + "```",
						},
						"position": {
							Type:        genai.TypeInteger,
							Description: "Use the line number that appears at the beginning of each line in the diff (such as 4, 5, 12, 13). These numbers are displayed with a colon (e.g., '4:', '5:').",
						},
					},
					PropertyOrdering: []string{"body", "position"},
				},
				Description: `Array of review comments.`,
			},
			"summary": {
				Type:        genai.TypeString,
				Description: "Write a summary for a single file being reviewed.",
				Example:     "Provides a concise English summary of the changes and issues found in the file under review.",
			},
		},
		Required: []string{"reviews", "summary"},
	}

	// Generate content
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  geminiApiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}

	maxOutputTokens := os.Getenv("MAX_OUTPUT_TOKENS")
	maxOutputTokensInt, err := strconv.Atoi(maxOutputTokens)
	if err != nil {
		maxOutputTokensInt = 2048
	}
	result, err := client.Models.GenerateContent(
		ctx,
		geminiModel,
		genai.Text(userPrompt),
		&genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
			MaxOutputTokens:  int32(maxOutputTokensInt),
			ResponseSchema:   &schema,
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: systemPrompt},
				},
				Role: "system",
			},
		},
	)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}
