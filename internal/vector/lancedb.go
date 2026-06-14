package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LanceStore talks to a local LanceDB sidecar over a small JSON/HTTP contract.
//
// Expected sidecar endpoints (table name is fixed to "cortex"):
//
//	POST {base}/upsert   body: vector.Record           -> 200
//	POST {base}/search   body: {vector, limit, filter} -> {matches: []Match}
//	POST {base}/delete   body: {id}                     -> 200
//
// A reference sidecar can be implemented in ~40 lines using the official
// LanceDB Python or TypeScript SDK. This keeps the Go daemon free of CGO and
// native build dependencies.
type LanceStore struct {
	BaseURL string
	HTTP    *http.Client
}

func NewLanceStore(baseURL string) *LanceStore {
	return &LanceStore{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *LanceStore) post(ctx context.Context, path string, in any, out any) error {
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("lancedb %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lancedb %s status %d: %s", path, resp.StatusCode, string(b))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *LanceStore) Upsert(ctx context.Context, rec Record) error {
	return s.post(ctx, "/upsert", rec, nil)
}

type searchReq struct {
	Vector []float32         `json:"vector"`
	Limit  int               `json:"limit"`
	Filter map[string]string `json:"filter,omitempty"`
}

type searchResp struct {
	Matches []Match `json:"matches"`
}

func (s *LanceStore) Search(ctx context.Context, vector []float32, limit int, filter map[string]string) ([]Match, error) {
	var out searchResp
	err := s.post(ctx, "/search", searchReq{Vector: vector, Limit: limit, Filter: filter}, &out)
	if err != nil {
		return nil, err
	}
	return out.Matches, nil
}

func (s *LanceStore) Delete(ctx context.Context, id string) error {
	return s.post(ctx, "/delete", map[string]string{"id": id}, nil)
}
