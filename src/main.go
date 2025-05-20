package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lee-lou2/ai-code-reviewer/cmd"
	"github.com/lee-lou2/ai-code-reviewer/pkg"
)

func main() {
	log.Println("🚀 Starting Go GitHub Action to comment on PR...")

	// Get PR
	ctx := context.Background()
	pr, err := pkg.GetPR(ctx)
	if err != nil {
		log.Fatalf("❌ Error: Failed to get PR: %v", err)
	}

	// Get README content
	readmeContent, err := pr.GetReadmeContent()
	if err != nil {
		log.Fatalf("❌ Error: Failed to get README content: %v", err)
	}

	// Get diff files
	files, err := pr.GetDiffFiles()
	if err != nil {
		pr.CreateIssueComment("❌ Error: Failed to get diff: " + err.Error())
		return
	}

	// Generate reviews
	summaries := "## 🌼 Summary\nReview results by file\n| File | Summary |\n| --- | --- |\n"
	for _, file := range files {
		reviews, err := cmd.GenReviews(ctx, pr, file, readmeContent)
		if err != nil {
			pr.CreateIssueComment("❌ Error: Failed to generate reviews: " + err.Error())
			return
		}

		// Create comments on PR
		for _, review := range reviews.Reviews {
			fmt.Printf("=== Review === %+v\n", review)
			if err := pr.CreatePRComments([]*pkg.Message{
				{
					Body:     review.Body,
					Path:     file.Path,
					Position: review.Position,
				},
			}); err != nil {
				pr.CreateIssueComment("❌ Error: Failed to create comments on PR: " + err.Error())
				return
			}
		}

		fmt.Printf("=== Summary === %+v\n", reviews.Summary)
		summaries += fmt.Sprintf("| %s | %s |\n", file.Path, reviews.Summary)
	}

	// Create comments on issue
	summaries += "\n"
	if err := pr.CreateIssueComment(summaries); err != nil {
		pr.CreateIssueComment("❌ Error: Failed to create comments on issue: " + err.Error())
		return
	}

	log.Printf("✅ Successfully created comments on PR #%d in %s/%s!", pr.Info.Number, pr.Info.Owner, pr.Info.Repo)
}
