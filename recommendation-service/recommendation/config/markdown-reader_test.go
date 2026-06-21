package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownReader_LoadMarkdown_ReturnsDocuments(t *testing.T) {
	reader := NewMarkdownReader()
	docs, err := reader.LoadMarkdown()

	assert.NoError(t, err)
	assert.NotEmpty(t, docs, "should return at least one document chunk")

	for i, doc := range docs {
		assert.NotEmpty(t, doc.Text, "document %d should have non-empty text", i)
		assert.NotEmpty(t, doc.Metadata["source"], "document %d should have source metadata", i)
	}
}

func TestMarkdownReader_LoadMarkdown_ContainsPriceTypes(t *testing.T) {
	reader := NewMarkdownReader()
	docs, err := reader.LoadMarkdown()

	assert.NoError(t, err)

	allText := ""
	for _, doc := range docs {
		allText += doc.Text + " "
	}

	assert.Contains(t, allText, "free")
	assert.Contains(t, allText, "half-price")
	assert.Contains(t, allText, "full-price")
	assert.Contains(t, allText, "Garden Pass")
	assert.Contains(t, allText, "Club Membership")
	assert.Contains(t, allText, "Patron Membership")
}

func TestMarkdownReader_LoadPrompt_ReturnsTemplate(t *testing.T) {
	reader := NewMarkdownReader()
	prompt, err := reader.LoadPrompt()

	assert.NoError(t, err)
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "{context}")
	assert.Contains(t, prompt, "{question}")
	assert.Contains(t, prompt, "CONTEXT")
}
