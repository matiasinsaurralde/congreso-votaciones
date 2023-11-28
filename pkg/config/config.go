package config

import (
	"encoding/json"
	"os"
)

// Config is the main configuration struct:
type Config struct {
	PDFPath      string              `json:"pdf_path"`
	ImagePath    string              `json:"image_path"`
	JSONPath     string              `json:"json_path"`
	StorePath    string              `json:"store_path"`
	SamplesPath  string              `json:"samples_path"`
	SampleData   map[string][]string `json:"sample_data"`
	OpenAIConfig OpenAIConfig        `json:"openai"`
}

// OpenAIConfig is the OpenAI configuration struct:
type OpenAIConfig struct {
	Token string `json:"token"`
}

// Load takes a file, parses it and returns a config:
func Load(fileName string) (*Config, error) {
	var cfg Config
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(contents, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
