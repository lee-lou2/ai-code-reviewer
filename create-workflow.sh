#!/bin/bash

# ANSI color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS="Linux";;
        Darwin*)    OS="Mac";;
        CYGWIN*|MINGW*|MSYS*)  OS="Windows";;
        *)          OS="Unknown";;
    esac
    echo -e "${BLUE}Detected OS: ${OS}${NC}"
}

# Create directory
create_directory() {
    # GitHub workflow directory path
    WORKFLOW_DIR=".github/workflows"
    
    # Create directory if it does not exist
    if [ ! -d "$WORKFLOW_DIR" ]; then
        echo -e "${BLUE}Creating directory: $WORKFLOW_DIR${NC}"
        mkdir -p "$WORKFLOW_DIR"
    else
        echo -e "${BLUE}Directory already exists: $WORKFLOW_DIR${NC}"
    fi
}

# Get user input
get_user_input() {
    # LANGUAGE input
    echo -e "${YELLOW}Set code review language (default: Korean)${NC}"
    echo -e "Examples: Korean, English, Japanese, Chinese"
    read -p "LANGUAGE: " LANGUAGE
    LANGUAGE=${LANGUAGE:-"Korean"}
    
    # GEMINI_MODEL input
    echo -e "\n${YELLOW}Set Gemini model (default: gemini-2.5-flash-preview-04-17)${NC}"
    echo -e "Examples: gemini-2.5-flash-preview-04-17, gemini-2.5-pro-preview-05-06"
    read -p "GEMINI_MODEL: " GEMINI_MODEL
    GEMINI_MODEL=${GEMINI_MODEL:-"gemini-2.5-flash-preview-04-17"}
    
    # OPENAI_MODEL input
    echo -e "\n${YELLOW}Set OpenAI model (default: gpt-4.1-mini)${NC}"
    echo -e "Examples: gpt-4.1-mini, gpt-4o, gpt-4o-mini"
    read -p "OPENAI_MODEL: " OPENAI_MODEL
    OPENAI_MODEL=${OPENAI_MODEL:-"gpt-4.1-mini"}

    # Max Output Tokens
    echo -e "\n${YELLOW}Set max output tokens (default: 2048)${NC}"
    read -p "MAX_OUTPUT_TOKENS: " MAX_OUTPUT_TOKENS
    MAX_OUTPUT_TOKENS=${MAX_OUTPUT_TOKENS:-"2048"}
    
    # EXCLUDE input
    echo -e "\n${YELLOW}Set exclude file patterns (default: *.md,*.txt,package-lock.json,*.yml,*.yaml)${NC}"
    echo -e "Examples: *.md,*.txt,*.json,node_modules/**,dist/**"
    read -p "EXCLUDE: " EXCLUDE
    EXCLUDE=${EXCLUDE:-"*.md,*.txt,package-lock.json,*.yml,*.yaml"}
}

# Create workflow file
create_workflow_file() {
    WORKFLOW_FILE=".github/workflows/ai-code-reviewer.yml"
    
    echo -e "${BLUE}Creating workflow file: $WORKFLOW_FILE${NC}"
    
    cat > "$WORKFLOW_FILE" << EOF
name: AI Code Reviewer

on:
  issue_comment:
    types: [created]

permissions: write-all

jobs:
  ai-code-review:
    runs-on: ubuntu-latest
    if: |
      github.event.issue.pull_request &&
      startsWith(github.event.comment.body, '/review')
    steps:
      - name: PR Info
        run: |
          echo "Comment: \${{ github.event.comment.body }}"
          echo "Issue Number: \${{ github.event.issue.number }}"
          echo "Repository: \${{ github.repository }}"

      - name: Checkout Repo
        uses: actions/checkout@v3
        with:
          fetch-depth: 0

      - name: Get PR Details
        id: pr
        run: |
          PR_JSON=\$(gh api repos/\${{ github.repository }}/pulls/\${{ github.event.issue.number }})
          echo "head_sha=\$(echo \$PR_JSON | jq -r .head.sha)" >> \$GITHUB_OUTPUT
          echo "base_sha=\$(echo \$PR_JSON | jq -r .base.sha)" >> \$GITHUB_OUTPUT
        env:
          GITHUB_TOKEN: \${{ secrets.GITHUB_TOKEN }}

      - uses: lee-lou2/ai-code-reviewer@main
        with:
          GITHUB_TOKEN: \${{ secrets.GITHUB_TOKEN }}
          GEMINI_API_KEY: \${{ secrets.GEMINI_API_KEY }}
          OPENAI_API_KEY: \${{ secrets.OPENAI_API_KEY }}
          LANGUAGE: $LANGUAGE
          GEMINI_MODEL: $GEMINI_MODEL
          OPENAI_MODEL: $OPENAI_MODEL
          MAX_OUTPUT_TOKENS: $MAX_OUTPUT_TOKENS
          EXCLUDE: "$EXCLUDE"
EOF

    echo -e "${GREEN}Workflow file successfully created: $WORKFLOW_FILE${NC}"
    echo -e "${YELLOW}Note: You must set GEMINI_API_KEY and OPENAI_API_KEY in GitHub secrets.${NC}"
}

# Main function
main() {
    echo -e "${GREEN}AI Code Reviewer Workflow Setup Script${NC}"
    echo -e "${BLUE}===========================================${NC}"
    
    detect_os
    create_directory
    get_user_input
    create_workflow_file
    
    echo -e "${GREEN}Setup complete!${NC}"
    echo -e "${BLUE}===========================================${NC}"
    echo -e "When you comment '/review' on a PR, the AI code review will start automatically."
}

# Run script
main
