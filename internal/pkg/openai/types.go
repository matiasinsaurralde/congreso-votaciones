package openai

import "encoding/json"

type CompletionRequest struct {
	Model          string                    `json:"model"`
	Messages       []Message                 `json:"messages"`
	MaxTokens      int                       `json:"max_tokens"`
	ResponseFormat *CompletionResponseFormat `json:"response_format,omitempty"`
}

type CompletionResponseFormat struct {
	Type string `json:"type"`
}

var CompletionResponseFormatJSON = CompletionResponseFormat{
	Type: "json_object",
}

func (c *CompletionRequest) ToJSON() ([]byte, error) {
	jsonObject, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return jsonObject, nil
}

type Message struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type ContentItem struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type CompletionResponse struct {
	ID      string                     `json:"id"`
	Choices []CompletionResponseChoice `json:"choices"`
}

func (c *CompletionResponse) FromJSON(rawJSON []byte) error {
	return json.Unmarshal(rawJSON, c)
}

type CompletionResponseChoice struct {
	Message CompletionResponseChoiceMessage `json:"message"`
}

type CompletionResponseChoiceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
