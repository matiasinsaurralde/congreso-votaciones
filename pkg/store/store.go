package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/matiasinsaurralde/congreso-votaciones/pkg/config"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/document"
	"github.com/matiasinsaurralde/congreso-votaciones/pkg/types"
	"github.com/rs/zerolog"
)

// Store implements a very basic data store for documents
// A vague locking mechanism is used by some methods:
type Store struct {
	cfg    *config.Config
	logger zerolog.Logger

	data *Data

	lock *sync.Mutex
}

// Data is the main store data structure
// When the store is initialized, it's loaded from disk -if a data file exists-:
type Data struct {
	Documents map[string]*document.Document `json:"documents"`
}

var (
	errDocumentNotFound = errors.New("document not found")
)

// Init initializes the store and loads existing data into memory:
func (s *Store) Init() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	// If data/data.json exists, pick it up
	// otherwise create it:
	if _, err := os.Stat(s.cfg.StorePath); err != nil {
		s.logger.Info().Msgf("Initializing store: %s", s.cfg.StorePath)
		if err := os.WriteFile(s.cfg.StorePath, []byte("{}"), 0755); err != nil {
			return err
		}
	}
	rawData, err := os.ReadFile(s.cfg.StorePath)
	if err != nil {
		return err
	}
	var storeData Data
	if err := json.Unmarshal(rawData, &storeData); err != nil {
		return err
	}
	s.logger.Info().Msgf("Store initialized from %s - %d bytes", s.cfg.StorePath, len(rawData))
	s.data = &storeData
	return nil
}

// AppendDocument adds a document to the store:
func (s *Store) AppendDocument(id string, d *document.Document) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.data.Documents == nil {
		s.data.Documents = make(map[string]*document.Document)
	}
	s.data.Documents[id] = d
	if err := s.save(); err != nil {
		return err
	}
	return nil
}

// RetrieveDocument retrieves a document from the store:
func (s *Store) RetrieveDocument(id string) *document.Document {
	s.lock.Lock()
	defer s.lock.Unlock()
	if doc, ok := s.data.Documents[id]; ok {
		return doc
	}
	return nil
}

// RetrieveDocumentsOpt sets options for filtering documents
// TODO: not implemented yet
type RetrieveDocumentsOpt struct{}

// RetrieveDocuments retrieves documents from the store:
func (s *Store) RetrieveDocuments(opts ...[]RetrieveDocumentsOpt) []*document.Document {
	s.lock.Lock()
	defer s.lock.Unlock()
	docs := make([]*document.Document, 0)
	for _, doc := range s.data.Documents {
		docs = append(docs, doc)
	}
	return docs
}

// GetDocumentCount returns the number of documents in the store:
func (s *Store) GetDocumentCount() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.data.Documents)
}

// save is an internal method for saving the current store data to disk:
func (s *Store) save() error {
	rawData, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.cfg.StorePath, rawData, 0755); err != nil {
		return err
	}
	return nil
}

// UpdateDocumentType updates the document type for a given document
// This is used by the classification step:
func (s *Store) UpdateDocumentType(id string, docType string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	doc, ok := s.data.Documents[id]
	if !ok {
		return errDocumentNotFound
	}
	doc.Type = types.DocumentType(docType)
	if err := s.save(); err != nil {
		return err
	}
	return nil
}

// New creates a new store with the given config and logger:
func New(cfg *config.Config, logger zerolog.Logger) *Store {
	s := &Store{
		lock:   &sync.Mutex{},
		cfg:    cfg,
		logger: logger,
	}
	return s
}
