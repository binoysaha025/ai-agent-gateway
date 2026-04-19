package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"github.com/binoysaha025/ai-agent-gateway/tools"
	"database/sql"
	"log"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Tools     []Tool    `json:"tools"`
	Messages  []Message `json:"messages"`
}

type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type ClaudeResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type ToolResult struct {
	Type      string 	`json:"type"`
	ToolUseID string     `json:"tool_use_id"`
	Content   string     `json:"content"`
}

func Run(prompt string, db *sql.DB) (string, int, error) {
	tools := getTools()
	messages := []Message{
		{Role: "user", Content: prompt},
	}

	totalTokens := 0

	// agent loop - max 5 iterations to prevent infinite loops
	for i := 0; i < 5; i++ {
		resp, err := callClaude(messages, tools)
		if err != nil {
			return "", 0, err
		}

		log.Printf("iteration %d: stop_reason=%s content_blocks=%d", i, resp.StopReason, len(resp.Content))
        for _, block := range resp.Content {
            log.Printf("  block type=%s name=%s", block.Type, block.Name)
        }

		totalTokens += resp.Usage.InputTokens + resp.Usage.OutputTokens

		// if Claude is done, return the text response
		if resp.StopReason == "end_turn" {
			for _, block := range resp.Content {
				if block.Type == "text" {
					return block.Text, totalTokens, nil
				}
			}
		}

		// if Claude wants to use a tool
		if resp.StopReason == "tool_use" {
			// add Claude's response to message history
			assistantContent, _ := json.Marshal(resp.Content)
			messages = append(messages, Message{
				Role:    "assistant",
				Content: string(assistantContent),
			})

			// execute each tool Claude requested
			toolResults := []ToolResult{}
			for _, block := range resp.Content {
				if block.Type == "tool_use" {
					result := executeTool(block.Name, block.Input, db)
					toolResults = append(toolResults, ToolResult{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content:   result,
					})
				}
			}

			// add tool results back to messages
			toolResultsJSON, _ := json.Marshal(toolResults)
			messages = append(messages, Message{
				Role:    "user",
				Content: string(toolResultsJSON),
			})
		}
	}

	return "", 0, fmt.Errorf("agent loop exceeded max iterations")
}

func callClaude(messages []Message, tools []Tool) (*ClaudeResponse, error) {
	apiKey := os.Getenv("ANTHROPIC_KEY")

	reqBody := ClaudeRequest{
		Model:     "claude-sonnet-4-6",
		MaxTokens: 1024,
		Tools:     tools,
		Messages:  messages,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var claudeResp ClaudeResponse
	err = json.Unmarshal(respBody, &claudeResp)
	if err != nil {
		return nil, err
	}
	log.Printf("claude raw response: %s", string(respBody))


	return &claudeResp, nil
}

func executeTool(name string, input json.RawMessage, db *sql.DB) string {
	switch name {
	case "calculate":
		return tools.Calculate(input)
	case "summarize_url":
		return tools.SummarizeURL(input)
	case "fetch_news":
		return tools.FetchNews(input)
	case "search_knowledge":
		return tools.SearchKnowledge(input, db)
	default:
		return "unknown tool"
	}
}

func getTools() []Tool {
	return []Tool{
		{
			Name:        "calculate",
			Description: "Evaluate a math expression. Use for any arithmetic or calculations.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"expression": {
						Type:        "string",
						Description: "Math expression to evaluate e.g. '42 * 7' or 'sqrt(144)'",
					},
				},
				Required: []string{"expression"},
			},
		},
		{
			Name:        "summarize_url",
			Description: "Fetch and extract text content from a URL. Use when asked to summarize or read a webpage.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"url": {
						Type:        "string",
						Description: "The full URL to fetch and summarize",
					},
				},
				Required: []string{"url"},
			},
		},
		{
			Name:        "fetch_news",
			Description: "Fetch latest top stories from Hacker News. Use when asked about tech news or trending topics.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"topic": {
						Type:        "string",
						Description: "Optional topic to filter news by e.g. 'AI' or 'golang'",
					},
				},
				Required: []string{},
			},
		},

		{
			Name:        "search_knowledge",
			Description: "Search the knowledge base for relevant context. Use when answering questions that might benefit from stored documents or domain knowledge.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
            "query": {
                Type:        "string",
                Description: "The search query to find relevant documents",
            },
        },
        Required: []string{"query"},

			}
		}
	}
}