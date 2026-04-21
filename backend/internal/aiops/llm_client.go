package aiops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LLMClient is an OpenAI-compatible HTTP client. No external Go deps required.
// Set OPENAI_API_KEY to enable. No-op (returns placeholder) when key is absent.
type LLMClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmRequest struct {
	Model       string       `json:"model"`
	Messages    []llmMessage `json:"messages"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
}

type llmResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewLLMClient() *LLMClient {
	return &LLMClient{
		apiKey:  os.Getenv("OPENAI_API_KEY"),
		baseURL: envOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		model:   envOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *LLMClient) available() bool { return c.apiKey != "" }

// Generate calls the LLM with a system + user prompt and returns the response text.
func (c *LLMClient) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.available() {
		return "[AI analysis disabled — set OPENAI_API_KEY to enable]", nil
	}

	payload := llmRequest{
		Model: c.model,
		Messages: []llmMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   600,
		Temperature: 0.2,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result llmResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("llm response parse failed: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("openai error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "[no response from AI model]", nil
	}
	return result.Choices[0].Message.Content, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
