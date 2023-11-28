package document

import (
	"encoding/base64"
	"os"

	"github.com/matiasinsaurralde/congreso-votaciones/pkg/types"
)

// Document is the main document struct:
type Document struct {
	// ID is currently the filename:
	ID string `json:"id"`
	// SourceURL is the URL where the document was downloaded from:
	SourceURL string `json:"source_url"`
	// ImagePaths is a list of paths to the rendered images:
	ImagePaths []string `json:"image_paths"`
	// PDFPath is the path to the downloaded PDF:
	PDFPath string `json:"pdf_path"`
	// JSONPath is the path to the JSON file containing the extracted data:
	JSONPath string `json:"json_path"`
	// Type is the document type -set during the classification step-:
	Type types.DocumentType `json:"type"`
}

// ImageAsBase64 returns the first image as a base64 string
// TODO: support page selection or implement an alternative method for it
func (d *Document) ImageAsBase64() (string, error) {
	// Pick the first page:
	image := d.ImagePaths[0]
	imageData, err := os.ReadFile(image)
	if err != nil {
		return "", err
	}
	encodedImage := base64.StdEncoding.EncodeToString(imageData)
	return encodedImage, nil
}
