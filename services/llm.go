package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AxeForging/releaseforge/helpers"
)

type LLMService struct {
	client *http.Client
}

func NewLLMService() *LLMService {
	return &LLMService{
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (l *LLMService) Generate(provider, apiKey, model, prompt string) (string, error) {
	helpers.Log.Info().Msgf("Initializing %s client with model: %s", provider, model)

	switch provider {
	case "gemini":
		return l.callGemini(apiKey, model, prompt)
	case "openai":
		return l.callOpenAI(apiKey, model, prompt)
	case "anthropic":
		return l.callAnthropic(apiKey, model, prompt)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (l *LLMService) callGemini(apiKey, model, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 8192,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", helpers.WrapError(err, "gemini", "failed to marshal request")
	}

	resp, err := l.client.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", helpers.WrapError(err, "gemini", "request failed")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", helpers.WrapError(err, "gemini", "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", helpers.WrapError(err, "gemini", "failed to parse response")
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (l *LLMService) callOpenAI(apiKey, model, prompt string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  8192,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", helpers.WrapError(err, "openai", "failed to marshal request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", helpers.WrapError(err, "openai", "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return "", helpers.WrapError(err, "openai", "request failed")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", helpers.WrapError(err, "openai", "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", helpers.WrapError(err, "openai", "failed to parse response")
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response")
	}

	return result.Choices[0].Message.Content, nil
}

func (l *LLMService) callAnthropic(apiKey, model, prompt string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"

	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 8192,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", helpers.WrapError(err, "anthropic", "failed to marshal request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", helpers.WrapError(err, "anthropic", "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", helpers.WrapError(err, "anthropic", "request failed")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", helpers.WrapError(err, "anthropic", "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", helpers.WrapError(err, "anthropic", "failed to parse response")
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty response")
	}

	return result.Content[0].Text, nil
}
