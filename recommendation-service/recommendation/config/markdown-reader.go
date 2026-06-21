package config

import (
	"strings"

	"service-app-go/recommendation-service/resources"
)

// Document represents a chunk of text loaded from the markdown knowledge base.
type Document struct {
	Text     string
	Metadata map[string]string
}

// MarkdownReader reads and chunks the embedded rag-text.md, mirroring the
// Spring MarkdownReader (MarkdownDocumentReader). Chunks are split by
// double-newline paragraphs.
type MarkdownReader struct{}

// NewMarkdownReader creates a new MarkdownReader.
func NewMarkdownReader() *MarkdownReader {
	return &MarkdownReader{}
}

// LoadMarkdown reads the embedded rag-text.md and splits it into Documents
// by paragraph (double newline). Each non-empty paragraph becomes one Document.
func (r *MarkdownReader) LoadMarkdown() ([]Document, error) {
	data, err := resources.RagText.ReadFile(resources.RagTextPath)
	if err != nil {
		return nil, err
	}

	text := string(data)
	paragraphs := strings.Split(text, "\n\n")

	docs := make([]Document, 0, len(paragraphs))
	for i, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		docs = append(docs, Document{
			Text:     trimmed,
			Metadata: map[string]string{"source": resources.RagTextPath, "chunk": string(rune('0' + i))},
		})
	}
	return docs, nil
}

// LoadPrompt reads the embedded rag-prompt.md template.
func (r *MarkdownReader) LoadPrompt() (string, error) {
	data, err := resources.RagPrompt.ReadFile(resources.RagPromptPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
