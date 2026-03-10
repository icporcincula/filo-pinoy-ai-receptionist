package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

// Qdrant client for RAG operations
type QdrantClient struct {
	baseURL    string
	apiKey     string
	collection string
	httpClient *http.Client
}

// Qdrant search request/response structures
type SearchRequest struct {
	CollectionName string      `json:"collection_name"`
	QueryVector    []float32   `json:"vector"`
	Limit          int         `json:"limit"`
	WithPayload    bool        `json:"with_payload"`
}

type SearchResponse struct {
	Result []struct {
		ID      interface{} `json:"id"`
		Score   float64     `json:"score"`
		Payload struct {
			Content string `json:"content"`
			Title   string `json:"title"`
			Source  string `json:"source"`
		} `json:"payload"`
	} `json:"result"`
	Status string `json:"status"`
}

// Document represents a business knowledge document
type Document struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Title   string `json:"title"`
	Source  string `json:"source"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// RAG system for business knowledge retrieval
type RAGSystem struct {
	qdrant *QdrantClient
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(baseURL, apiKey, collection string) *QdrantClient {
	return &QdrantClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		collection: collection,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewRAGSystem creates a new RAG system
func NewRAGSystem(qdrant *QdrantClient) *RAGSystem {
	return &RAGSystem{
		qdrant: qdrant,
	}
}

// Search performs semantic search in the knowledge base
func (r *RAGSystem) Search(ctx context.Context, query string, limit int) ([]Document, error) {
	// For now, we'll use a simple keyword-based search simulation
	// In a real implementation, this would involve embedding the query
	// and performing vector similarity search in Qdrant
	
	// Simulate embedding the query (in real implementation, use an embedding model)
	queryEmbedding := simulateEmbedding(query)
	
	searchReq := SearchRequest{
		CollectionName: r.qdrant.collection,
		QueryVector:    queryEmbedding,
		Limit:          limit,
		WithPayload:    true,
	}
	
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", 
		fmt.Sprintf("%s/points/search", r.qdrant.baseURL), 
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if r.qdrant.apiKey != "" {
		req.Header.Set("api-key", r.qdrant.apiKey)
	}
	
	resp, err := r.qdrant.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search request failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	var searchResp SearchResponse
	err = json.Unmarshal(respBody, &searchResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal search response: %w", err)
	}
	
	documents := make([]Document, len(searchResp.Result))
	for i, result := range searchResp.Result {
		documents[i] = Document{
			ID:      fmt.Sprintf("%v", result.ID),
			Content: result.Payload.Content,
			Title:   result.Payload.Title,
			Source:  result.Payload.Source,
		}
	}
	
	return documents, nil
}

// AddDocument adds a document to the knowledge base
func (r *RAGSystem) AddDocument(ctx context.Context, doc Document) error {
	// Simulate embedding the document content
	contentEmbedding := simulateEmbedding(doc.Content)
	
	point := struct {
		ID      string                 `json:"id"`
		Vector  []float32             `json:"vector"`
		Payload map[string]interface{} `json:"payload"`
	}{
		ID:     doc.ID,
		Vector: contentEmbedding,
		Payload: map[string]interface{}{
			"content": doc.Content,
			"title":   doc.Title,
			"source": doc.Source,
		},
	}
	
	points := struct {
		CollectionName string      `json:"collection_name"`
		Points         interface{} `json:"points"`
	}{
		CollectionName: r.qdrant.collection,
		Points:         []interface{}{point},
	}
	
	reqBody, err := json.Marshal(points)
	if err != nil {
		return fmt.Errorf("failed to marshal point: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "PUT", 
		fmt.Sprintf("%s/points?wait=true", r.qdrant.baseURL), 
		bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create add point request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if r.qdrant.apiKey != "" {
		req.Header.Set("api-key", r.qdrant.apiKey)
	}
	
	resp, err := r.qdrant.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read add response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("add document request failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	return nil
}

// simulateEmbedding creates a simulated embedding for demonstration purposes
// In a real implementation, this would call an embedding model
func simulateEmbedding(text string) []float32 {
	// This is a placeholder implementation that creates a simple hash-based embedding
	// In a real implementation, you would use a proper embedding model like Sentence Transformers
	embedding := make([]float32, 1536) // Typical size for embeddings
	
	// Simple hash-based approach for demonstration
	for i := 0; i < len(text) && i < len(embedding); i++ {
		embedding[i] = float32(text[i]) / 25.0
	}
	
	// Normalize the embedding
	sum := float32(0)
	for _, v := range embedding {
		sum += v * v
	}
	sum = float32(sum)
	if sum > 0 {
		sum = float32(1.0 / math.Sqrt(float64(sum)))
		for i := range embedding {
			embedding[i] *= sum
		}
	}
	
	return embedding
}

// GetRAGContext retrieves relevant context for a query
func (r *RAGSystem) GetRAGContext(ctx context.Context, query string) (string, error) {
	docs, err := r.Search(ctx, query, 3) // Get top 3 relevant documents
	if err != nil {
		return "", fmt.Errorf("failed to search knowledge base: %w", err)
	}
	
	if len(docs) == 0 {
		return "", nil // No relevant documents found
	}
	
	var contextBuilder bytes.Buffer
	contextBuilder.WriteString("Business Knowledge Base Information:\n\n")
	
	for i, doc := range docs {
		contextBuilder.WriteString(fmt.Sprintf("Source %d (%s): %s\n", i+1, doc.Title, doc.Content))
		if i < len(docs)-1 {
			contextBuilder.WriteString("\n")
		}
	}
	
	return contextBuilder.String(), nil
}