package openai

import (
	"context"
	"fmt"

	goopenai "github.com/sashabaranov/go-openai"

	"zhiyuwaf/internal/ai"
)

type Client struct {
	client *goopenai.Client
	model  string
}

func NewClient(apiKey, model, baseURL string) *Client {
	cfg := goopenai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	if model == "" {
		model = goopenai.GPT4o
	}

	return &Client{
		client: goopenai.NewClientWithConfig(cfg),
		model:  model,
	}
}

func (c *Client) Name() string { return "openai" }

func (c *Client) Analyze(ctx context.Context, req ai.AnalysisRequest) (*ai.AnalysisResponse, error) {
	systemPrompt, userMsg := ai.BuildPrompt(req)

	resp, err := c.client.CreateChatCompletion(ctx, goopenai.ChatCompletionRequest{
		Model: c.model,
		Messages: []goopenai.ChatCompletionMessage{
			{Role: goopenai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: goopenai.ChatMessageRoleUser, Content: userMsg},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("openai API call: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from openai")
	}

	content := resp.Choices[0].Message.Content
	return ai.ParseResponse(content)
}
