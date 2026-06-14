// Package vector defines the semantic-search abstraction for CORTEX.
//
// LanceDB has no official Go SDK (only Python, TypeScript and Rust), so the
// vector store is consumed over a small HTTP contract served by a local
// LanceDB sidecar. The Store interface keeps this decoupled: swapping the
// sidecar for a CGO binding later only requires a new Store implementation.
package vector

import "context"

// Record is a single embedded item stored in the vector index.
type Record struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	Collection string            `json:"collection"` // file | vault | function | task | decision
	Text       string            `json:"text"`
	Vector     []float32         `json:"vector"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Match is a single nearest-neighbour search hit.
type Match struct {
	ID         string  `json:"id"`
	ProjectID  string  `json:"project_id"`
	Collection string  `json:"collection"`
	Text       string  `json:"text"`
	Score      float64 `json:"score"`
}

// Store is the minimal vector index surface CORTEX needs.
type Store interface {
	Upsert(ctx context.Context, rec Record) error
	Search(ctx context.Context, vector []float32, limit int, filter map[string]string) ([]Match, error)
	Delete(ctx context.Context, id string) error
}

// Embedder turns text into an embedding vector.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}
