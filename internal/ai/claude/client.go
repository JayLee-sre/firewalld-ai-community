package claude

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"zhiyuwaf/internal/ai"
)

type Client struct {
	client anthropic.Client
	model  string
}

func NewClient(apiKey, model, baseURL string) *Client {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := anthropic.NewClient(opts...)

	if model == "" {
		model = string(anthropic.ModelClaudeSonnet4_20250514)
	}

	return &Client{
		client: client,
		model:  model,
	}
}

func (c *Client) Name() string { return "claude" }

func (c *Client) Analyze(ctx context.Context, req ai.AnalysisRequest) (*ai.AnalysisResponse, error) {
	systemPrompt, userMsg := ai.BuildPrompt(req)

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude API call: %w", err)
	}

	if len(message.Content) == 0 {
		return nil, fmt.Errorf("empty response from claude")
	}

	content := message.Content[0].Text
	return ai.ParseResponse(content)
}
