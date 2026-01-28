package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// callAnthropic calls the Anthropic API for tool selection
func (s *SearchService) callAnthropic(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	s.logger.Debug("Calling Anthropic API", zap.String("model", s.samplingCfg.Model))

	// Prepare request body
	requestBody := map[string]interface{}{
		"model":      s.samplingCfg.Model,
		"max_tokens": 2000,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"system": systemPrompt,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.samplingCfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Make request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResponse.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", apiResponse.Error.Message)
	}

	if len(apiResponse.Content) == 0 {
		return "", fmt.Errorf("no content in API response")
	}

	s.logger.Debug("Anthropic API call successful")
	return apiResponse.Content[0].Text, nil
}

// callOpenAI calls the OpenAI API for tool selection
func (s *SearchService) callOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	s.logger.Debug("Calling OpenAI API", zap.String("model", s.samplingCfg.Model))

	// Prepare request body
	requestBody := map[string]interface{}{
		"model": s.samplingCfg.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"max_tokens":  2000,
		"temperature": 0.7,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.samplingCfg.APIKey))

	// Make request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResponse.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", apiResponse.Error.Message)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices in API response")
	}

	s.logger.Debug("OpenAI API call successful")
	return apiResponse.Choices[0].Message.Content, nil
}
