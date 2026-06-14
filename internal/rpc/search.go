package rpc

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/NexVed/Cortex/internal/search"
	"github.com/rs/zerolog/log"
)

func (h *Handlers) Search(ctx context.Context, req *connect.Request[v1.SearchRequest]) (*connect.Response[v1.SearchResponse], error) {
	results, err := h.SearchEngine.Search(search.Query{
		Query:     req.Msg.Query,
		ProjectID: req.Msg.ProjectId,
		Scope:     req.Msg.Scope,
		Limit:     int(req.Msg.Limit),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.SearchResponse{}
	for _, r := range results {
		out.Results = append(out.Results, &v1.SearchResult{
			Id:         r.ID,
			Collection: r.Collection,
			ProjectId:  r.ProjectID,
			Title:      r.Title,
			Excerpt:    r.Excerpt,
		})
	}
	h.SearchEngine.RecordHistory(userID(ctx), search.Query{Query: req.Msg.Query, Scope: req.Msg.Scope}, len(out.Results))
	return connect.NewResponse(out), nil
}

func (h *Handlers) SemanticSearch(ctx context.Context, req *connect.Request[v1.SemanticSearchRequest]) (*connect.Response[v1.SemanticSearchResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 20
	}

	// Semantic search requires both an embedder and a configured vector store.
	// When either is unavailable, fall back to keyword search so the endpoint
	// degrades gracefully rather than failing.
	if h.Embedder == nil || h.Vector == nil {
		log.Debug().Msg("semantic search not configured; falling back to keyword search")
		return h.semanticFallback(req, limit)
	}

	vec, err := h.Embedder.Embed(ctx, req.Msg.Query)
	if err != nil {
		log.Warn().Err(err).Msg("embedding failed; falling back to keyword search")
		return h.semanticFallback(req, limit)
	}

	filter := map[string]string{}
	if req.Msg.ProjectId != "" {
		filter["project_id"] = req.Msg.ProjectId
	}
	matches, err := h.Vector.Search(ctx, vec, limit, filter)
	if err != nil {
		log.Warn().Err(err).Msg("vector search failed; falling back to keyword search")
		return h.semanticFallback(req, limit)
	}

	out := &v1.SemanticSearchResponse{}
	for _, m := range matches {
		out.Results = append(out.Results, &v1.SearchResult{
			Id:         m.ID,
			Collection: m.Collection,
			ProjectId:  m.ProjectID,
			Excerpt:    m.Text,
			Score:      m.Score,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *Handlers) semanticFallback(req *connect.Request[v1.SemanticSearchRequest], limit int) (*connect.Response[v1.SemanticSearchResponse], error) {
	results, err := h.SearchEngine.Search(search.Query{
		Query:     req.Msg.Query,
		ProjectID: req.Msg.ProjectId,
		Limit:     limit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.SemanticSearchResponse{}
	for _, r := range results {
		out.Results = append(out.Results, &v1.SearchResult{
			Id:         r.ID,
			Collection: r.Collection,
			ProjectId:  r.ProjectID,
			Title:      r.Title,
			Excerpt:    r.Excerpt,
		})
	}
	return connect.NewResponse(out), nil
}
