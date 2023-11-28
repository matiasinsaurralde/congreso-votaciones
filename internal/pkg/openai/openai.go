package openai

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/matiasinsaurralde/congreso-votaciones/pkg/config"
)

var (
	errNoTokenSet = errors.New("no token set")
)

// OAIClient wraps OpenAI API calls:
type OAIClient struct {
	cfg *config.Config
}

// Completion calls the chat completion endpoint: https://platform.openai.com/docs/guides/text-generation/chat-completions-api
func (c *OAIClient) Completion(completionRequest *CompletionRequest) (*CompletionResponse, error) {
	if c.cfg.OpenAIConfig.Token == "" {
		return nil, errNoTokenSet
	}

	if completionRequest.Model == "" {
		completionRequest.Model = defaultModel
	}

	reqJSON, err := completionRequest.ToJSON()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		completionEndPoint,
		bytes.NewReader(reqJSON),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.OpenAIConfig.Token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if req.Body == nil {
		return nil, errors.New("nil body")
	}

	// TODO: handle in a proper way
	if res.StatusCode == http.StatusTooManyRequests {
		time.Sleep(60 * time.Second)
		return nil, fmt.Errorf("too many requests: %s", res.Status)
	}
	rawBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var completionResponse CompletionResponse
	if err := completionResponse.FromJSON(rawBody); err != nil {
		return nil, err
	}
	return &completionResponse, nil
}

// New initializes a new OpenAI API client:
func New(cfg *config.Config) *OAIClient {
	return &OAIClient{
		cfg: cfg,
	}
}
