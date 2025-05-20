package pkg

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// GenOpenAI generates reviews for a file using OpenAI
func GenOpenAI(ctx context.Context, userPrompt, systemPrompt string) (string, error) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	openaiModel := os.Getenv("OPENAI_MODEL")
	if openaiAPIKey == "" || openaiModel == "" {
		return "", fmt.Errorf("required OPENAI_API_KEY or OPENAI_MODEL environment variable is not set")
	}

	// Generate schema for the response
	schema, err := jsonschema.GenerateSchemaForType(struct {
		Reviews []struct {
			Body     string `json:"body"`
			Position int    `json:"position"`
		} `json:"reviews"`
		Summary string `json:"summary"`
	}{})
	if err != nil {
		return "", err
	}

	// Generate reviews
	client := openai.NewClient(openaiAPIKey)
	maxOutputTokens := os.Getenv("MAX_OUTPUT_TOKENS")
	maxOutputTokensInt, err := strconv.Atoi(maxOutputTokens)
	if err != nil {
		maxOutputTokensInt = 2048
	}
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     openaiModel,
		MaxTokens: maxOutputTokensInt,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "Reviews",
				Schema: schema,
				Strict: true,
			},
		},
	})
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
