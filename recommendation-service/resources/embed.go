package resources

import "embed"

//go:embed document/rag-text.md
var RagText embed.FS

//go:embed prompts/rag-prompt.md
var RagPrompt embed.FS

// RagTextPath is the path to the knowledge base markdown inside the embed FS.
const RagTextPath = "document/rag-text.md"

// RagPromptPath is the path to the prompt template inside the embed FS.
const RagPromptPath = "prompts/rag-prompt.md"
