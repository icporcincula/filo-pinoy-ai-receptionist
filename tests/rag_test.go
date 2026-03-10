package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock HTTP client for testing
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// Test QdrantClient functionality
func TestQdrantClient(t *testing.T) {
	t.Run("NewQdrantClient", func(t *testing.T) {
		client := NewQdrantClient("http://localhost:6333", "test-key", "test-collection")
		assert.NotNil(t, client)
		assert.Equal(t, "http://localhost:6333", client.baseURL)
		assert.Equal(t, "test-key", client.apiKey)
		assert.Equal(t, "test-collection", client.collection)
		assert.NotNil(t, client.httpClient)
	})

	t.Run("Search", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/points/search", r.URL.Path)
			
			// Verify request body
			body, _ := io.ReadAll(r.Body)
			var searchReq SearchRequest
			err := json.Unmarshal(body, &searchReq)
			assert.NoError(t, err)
			assert.Equal(t, "test-collection", searchReq.CollectionName)
			assert.Equal(t, 3, searchReq.Limit)
			assert.True(t, searchReq.WithPayload)
			
			// Return mock response
			response := SearchResponse{
				Result: []struct {
					ID      interface{} `json:"id"`
					Score   float64     `json:"score"`
					Payload struct {
						Content string `json:"content"`
						Title   string `json:"title"`
						Source  string `json:"source"`
					} `json:"payload"`
				}{
					{
						ID:    "doc1",
						Score: 0.95,
						Payload: struct {
							Content string `json:"content"`
							Title   string `json:"title"`
							Source  string `json:"source"`
						}{
							Content: "Test business content",
							Title:   "Test Document",
							Source:  "test.txt",
						},
					},
				},
				Status: "ok",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		client := NewQdrantClient(mockServer.URL, "test-key", "test-collection")
		ctx := context.Background()

		docs, err := client.Search(ctx, "test query", 3)
		assert.NoError(t, err)
		assert.Len(t, docs, 1)
		assert.Equal(t, "doc1", docs[0].ID)
		assert.Equal(t, "Test business content", docs[0].Content)
		assert.Equal(t, "Test Document", docs[0].Title)
		assert.Equal(t, "test.txt", docs[0].Source)
	})

	t.Run("AddDocument", func(t *testing.T) {
		// Create mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/points?wait=true", r.URL.Path)
			
			// Verify request body
			body, _ := io.ReadAll(r.Body)
			var points struct {
				CollectionName string      `json:"collection_name"`
				Points         interface{} `json:"points"`
			}
			err := json.Unmarshal(body, &points)
			assert.NoError(t, err)
			assert.Equal(t, "test-collection", points.CollectionName)
			
			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer mockServer.Close()

		client := NewQdrantClient(mockServer.URL, "test-key", "test-collection")
		ctx := context.Background()

		doc := Document{
			ID:      "test-doc",
			Content: "Test document content",
			Title:   "Test Title",
			Source:  "test.txt",
		}

		err := client.AddDocument(ctx, doc)
		assert.NoError(t, err)
	})
}

// Test RAGSystem functionality
func TestRAGSystem(t *testing.T) {
	t.Run("NewRAGSystem", func(t *testing.T) {
		qdrantClient := NewQdrantClient("http://localhost:6333", "test-key", "test-collection")
		ragSystem := NewRAGSystem(qdrantClient)
		assert.NotNil(t, ragSystem)
		assert.Equal(t, qdrantClient, ragSystem.qdrant)
	})

	t.Run("GetRAGContext", func(t *testing.T) {
		// Create mock server that returns search results
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := SearchResponse{
				Result: []struct {
					ID      interface{} `json:"id"`
					Score   float64     `json:"score"`
					Payload struct {
						Content string `json:"content"`
						Title   string `json:"title"`
						Source  string `json:"source"`
					} `json:"payload"`
				}{
					{
						ID:    "doc1",
						Score: 0.95,
						Payload: struct {
							Content string `json:"content"`
							Title   string `json:"title"`
							Source  string `json:"source"`
						}{
							Content: "We are open from 9:00 AM to 6:00 PM, Monday through Friday.",
							Title:   "Business Hours",
							Source:  "faq.txt",
						},
					},
					{
						ID:    "doc2", 
						Score: 0.85,
						Payload: struct {
							Content string `json:"content"`
							Title   string `json:"title"`
							Source  string `json:"source"`
						}{
							Content: "You can book an appointment by speaking to me, and I'll help you find available time slots.",
							Title:   "Appointment Booking",
							Source:  "services.txt",
						},
					},
				},
				Status: "ok",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		qdrantClient := NewQdrantClient(mockServer.URL, "test-key", "test-collection")
		ragSystem := NewRAGSystem(qdrantClient)
		ctx := context.Background()

		context, err := ragSystem.GetRAGContext(ctx, "What are your business hours?")
		assert.NoError(t, err)
		assert.Contains(t, context, "Business Knowledge Base Information:")
		assert.Contains(t, context, "We are open from 9:00 AM to 6:00 PM")
		assert.Contains(t, context, "You can book an appointment")
	})

	t.Run("GetRAGContext with no results", func(t *testing.T) {
		// Create mock server that returns empty results
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := SearchResponse{
				Result: []struct {
					ID      interface{} `json:"id"`
					Score   float64     `json:"score"`
					Payload struct {
						Content string `json:"content"`
						Title   string `json:"title"`
						Source  string `json:"source"`
					} `json:"payload"`
				}{},
				Status: "ok",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		qdrantClient := NewQdrantClient(mockServer.URL, "test-key", "test-collection")
		ragSystem := NewRAGSystem(qdrantClient)
		ctx := context.Background()

		context, err := ragSystem.GetRAGContext(ctx, "Unknown query")
		assert.NoError(t, err)
		assert.Empty(t, context)
	})
}

// Test embedding simulation
func TestSimulateEmbedding(t *testing.T) {
	t.Run("Basic embedding", func(t *testing.T) {
		text := "Hello world"
		embedding := simulateEmbedding(text)
		
		assert.NotNil(t, embedding)
		assert.Len(t, embedding, 1536) // Standard embedding size
		
		// Check that embedding is normalized
		sum := float32(0)
		for _, v := range embedding {
			sum += v * v
		}
		assert.InDelta(t, 1.0, sum, 0.001) // Should be normalized to 1
	})

	t.Run("Different text produces different embeddings", func(t *testing.T) {
		embedding1 := simulateEmbedding("Hello world")
		embedding2 := simulateEmbedding("Goodbye world")
		
		assert.NotEqual(t, embedding1, embedding2)
	})

	t.Run("Empty text handling", func(t *testing.T) {
		embedding := simulateEmbedding("")
		assert.NotNil(t, embedding)
		assert.Len(t, embedding, 1536)
		// Should be all zeros for empty input
		for _, v := range embedding {
			assert.Equal(t, float32(0), v)
		}
	})
}

// Benchmark tests
func BenchmarkRAGSystem_Search(b *testing.B) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SearchResponse{
			Result: []struct {
				ID      interface{} `json:"id"`
				Score   float64     `json:"score"`
				Payload struct {
					Content string `json:"content"`
					Title   string `json:"title"`
					Source  string `json:"source"`
				} `json:"payload"`
			}{
				{
					ID:    "doc1",
					Score: 0.95,
					Payload: struct {
						Content string `json:"content"`
						Title   string `json:"title"`
						Source  string `json:"source"`
					}{
						Content: "Test content",
						Title:   "Test",
						Source:  "test.txt",
					},
				},
			},
			Status: "ok",
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	qdrantClient := NewQdrantClient(mockServer.URL, "test-key", "test-collection")
	ragSystem := NewRAGSystem(qdrantClient)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ragSystem.GetRAGContext(ctx, "test query")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimulateEmbedding(b *testing.B) {
	text := "This is a test sentence for embedding generation"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = simulateEmbedding(text)
	}
}