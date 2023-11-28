package processor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matiasinsaurralde/congreso-votaciones/internal/pkg/openai"
	"github.com/matiasinsaurralde/congreso-votaciones/internal/pkg/pdf2png"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/config"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/document"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/store"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Processor wraps the document processing logic:
type Processor struct {
	// cfg is the main configuration:
	cfg *config.Config
	// store is the main store:
	store *store.Store
	// logger is the main logger:
	logger zerolog.Logger
	// oaiClient is the OpenAI API client:
	oaiClient *openai.OAIClient
	// samples is a map of label -> sample documents:
	samples map[string][]*document.Document
}

// ClassificationOutput is the output of the classification step
// The current prompt enforces the usage of this structure:
type ClassificationOutput struct {
	// Label is the label of the sample that matched the document
	// so it corresponds to the key in the sample data map:
	Label string `json:"label"`
	// Similar is a boolean that indicates if the document is similar
	// For simplicity we don't currently ask for a confidence score:
	Similar bool `json:"similar"`
}

// loadSamples loads the sample data from the configuration and
// generates a rendered image for each one. This is required
// as OpenAI doesn't currently take PDF input:
func (p *Processor) loadSamples() error {
	p.logger.Info().Msg("Loading samples")
	sampleCount := 0
	for label, sampleDocs := range p.cfg.SampleData {
		if p.samples[label] == nil {
			p.samples[label] = make([]*document.Document, 0)
		}
		for _, samplePath := range sampleDocs {
			p.logger.Debug().Msgf("Loading sample %s", samplePath)
			fullPath := filepath.Join(p.cfg.SamplesPath, samplePath)
			t := types.DocumentType(label)
			doc := document.Document{
				SourceURL: fullPath,
				PDFPath:   fullPath,
				Type:      t,
			}
			p.logger.Debug().Msgf("Generating image for %s - type %v", fullPath, t)
			if err := p.generateSampleImage(&doc); err != nil {
				return err
			}
			p.samples[label] = append(p.samples[label], &doc)
			sampleCount++
		}
	}
	p.logger.Info().Msgf("Loaded %d samples", sampleCount)
	return nil
}

// generateSampleImage generates a sample image for a given document
func (p *Processor) generateSampleImage(d *document.Document) error {
	fileName := filepath.Base(d.SourceURL)
	newFileName := strings.ReplaceAll(fileName, ".pdf", ".png")
	newFilePath := filepath.Join(p.cfg.ImagePath, newFileName)
	err := pdf2png.RenderPage(d.SourceURL, newFilePath, 0)
	if err != nil {
		return err
	}
	d.ImagePaths = append(d.ImagePaths, newFilePath)
	return nil
}

// loadDocuments loads the documents from the PDFs path:
func (p *Processor) loadDocuments() error {
	p.logger.Info().Msg("Loading documents")
	filepath.WalkDir(p.cfg.PDFPath, func(path string, d os.DirEntry, err error) error {
		// Skip all non-PDF files:
		if filepath.Ext(path) != ".pdf" {
			return nil
		}
		fileName := filepath.Base(path)

		// First check if document exists or not:
		if doc := p.store.RetrieveDocument(fileName); doc != nil {
			p.logger.Debug().Msgf("Document %s already exists - skipping", fileName)
			return nil
		}

		newFileName := strings.ReplaceAll(fileName, ".pdf", ".png")
		newFilePath := filepath.Join(p.cfg.ImagePath, newFileName)
		doc := document.Document{
			ID:        fileName,
			SourceURL: path,
			PDFPath:   path,
		}

		pageCount, err := pdf2png.GetPageCount(path)
		if err != nil {
			return err
		}

		// Handle both single page and multi page scenarios:
		if pageCount == 1 {
			if err := pdf2png.RenderPage(path, newFilePath, 0); err != nil {
				return err
			}
			doc.ImagePaths = []string{newFilePath}
		} else {
			count := pageCount
			imagePaths := make([]string, 0)

			// When rendering multi page input use page count as a suffix in every rendered image:
			for i := 0; i < count; i++ {
				newFilePath := strings.ReplaceAll(newFilePath, ".png", fmt.Sprintf("_%d.png", i))
				if err := pdf2png.RenderPage(path, newFilePath, i); err != nil {
					return err
				}
				imagePaths = append(imagePaths, newFilePath)
			}
			doc.ImagePaths = imagePaths
		}

		// Store the updated document data:
		if err := p.store.AppendDocument(doc.ID, &doc); err != nil {
			return err
		}
		return nil
	})
	p.logger.Info().Msgf("Loaded %d documents", p.store.GetDocumentCount())
	return nil
}

// classifyDocument classifies a document using the OpenAI API
// It's currently very inefficient due to API rate limiting
// Using batch requests should improve it:
func (p *Processor) classifyDocument(d *document.Document) (*ClassificationOutput, error) {
	docImage, err := d.ImageAsBase64()
	if err != nil {
		return nil, err
	}
	for label, samples := range p.samples {
		sample := samples[0]
		p.logger.Debug().Msgf("Comparing '%s' with sample '%s'", filepath.Base(d.ID), label)
		sampleImageB64, _ := sample.ImageAsBase64()
		completionRequest := openai.CompletionRequest{
			MaxTokens: 3000,
			Messages: []openai.Message{
				{Role: "user", Content: []openai.ContentItem{
					{
						Type: "text",
						Text: `
Analyze the layout and format of the two images.
If the images are highly similar, return a JSON object with the following structure:
{"simiar": true}
If the images are not similar return:
{"similar": false}
Don't return any more output than JSON.
`,
					},
					{
						Type: "image_url",
						ImageURL: &openai.ImageURL{
							URL: "data:image/png;base64," + docImage,
						},
					},
					{
						Type: "image_url",
						ImageURL: &openai.ImageURL{
							URL: "data:image/png;base64," + sampleImageB64,
						},
					},
				}},
			},
		}
		res, err := p.oaiClient.Completion(&completionRequest)
		if err != nil {
			return nil, err
		}
		if len(res.Choices) == 0 {
			return nil, errors.New("no choices returned from completion API")
		}

		// Take the first/only choice and parse the output:
		classification, err := p.getClassification(res.Choices[0])
		if err != nil {
			log.Err(err).Msg("error getting classification")
			return nil, err
		}

		// If similar return earlier and avoid further processing against other samples:
		if classification.Similar {
			classification.Label = label
			return classification, nil
		}
	}

	return nil, nil
}

// getClassification parses the classification output from the OpenAI API:
func (p *Processor) getClassification(choice openai.CompletionResponseChoice) (*ClassificationOutput, error) {
	jsonBlock := choice.Message.Content
	if len(jsonBlock) == 0 {
		return nil, errors.New("no content found")
	}

	// Do some basic checks to make sure we're parsing the right thing:
	if strings.Contains(jsonBlock, "```json") {
		splits := strings.Split(jsonBlock, "```json")
		if len(splits) == 0 {
			return nil, errors.New("no JSON code block found")
		}
		split := splits[1]
		jsonBlock = strings.Split(split, "```")[0]
	}

	// Actually unmarshal the JSON block:
	var output ClassificationOutput
	reader := strings.NewReader(jsonBlock)
	if err := json.NewDecoder(reader).Decode(&output); err != nil {
		return nil, err
	}
	return &output, nil
}

// Classify is the high level classification step:
func (p *Processor) Classify() error {
	// Load samples into memory:
	if err := p.loadSamples(); err != nil {
		return err
	}

	// Load documents into memory:
	if err := p.loadDocuments(); err != nil {
		return err
	}

	// Loop forever?
	for {
		for _, d := range p.store.RetrieveDocuments() {
			if d.Type != "" {
				p.logger.Info().Msgf("skipping %s - already classified", d.ID)
				continue
			}
			ts := time.Now()
			baseName := filepath.Base(d.PDFPath)
			p.logger.Info().Msgf("classifying %s", baseName)
			classification, err := p.classifyDocument(d)
			if err != nil {
				p.logger.Err(err).Msg("error classifying document")
				continue
			}
			diff := time.Since(ts)
			p.logger.Info().Msgf("done: %+v - took %d seconds", classification, diff.Milliseconds())

			// Update store:
			if err := p.store.UpdateDocumentType(d.ID, classification.Label); err != nil {
				return err
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// Parse is the high level parsing step:
func (p *Processor) Parse() error {
	return nil
}

func (p *Processor) Transform() error {
	return nil
}

// New initializes a new processor with the given components:
func New(cfg *config.Config, store *store.Store, logger zerolog.Logger) *Processor {
	p := &Processor{
		samples:   make(map[string][]*document.Document),
		oaiClient: openai.New(cfg),
		cfg:       cfg,
		store:     store,
		logger:    logger,
	}
	return p
}
