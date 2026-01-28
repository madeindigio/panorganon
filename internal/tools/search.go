package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sevir/panorganon/internal/database"
	"go.uber.org/zap"
)

// SearchResult represents a single tool search result with relevance score
type SearchResult struct {
	Name        string                 `json:"name"`
	Server      string                 `json:"server"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Score       float64                `json:"score"`
}

// SearchService handles intelligent tool search using LLM sampling
type SearchService struct {
	db           *database.DB
	logger       *zap.Logger
	samplingCfg  SamplingConfig
	cache        map[string]*cachedSearch
	cacheMux     sync.RWMutex
	cacheTTL     time.Duration
}

// SamplingConfig holds LLM sampling configuration
type SamplingConfig struct {
	Provider string
	APIKey   string
	Model    string
}

// cachedSearch represents a cached search result
type cachedSearch struct {
	Results   []SearchResult
	Timestamp time.Time
}

// NewSearchService creates a new search service
func NewSearchService(db *database.DB, logger *zap.Logger, cfg SamplingConfig) *SearchService {
	return &SearchService{
		db:          db,
		logger:      logger.With(zap.String("component", "search")),
		samplingCfg: cfg,
		cache:       make(map[string]*cachedSearch),
		cacheTTL:    5 * time.Minute,
	}
}

// SearchTools searches for tools based on a task description using LLM sampling
func (s *SearchService) SearchTools(ctx context.Context, taskDescription string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	s.logger.Info("Searching tools",
		zap.String("task", taskDescription),
		zap.Int("max_results", maxResults),
	)

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%d", taskDescription, maxResults)
	if cached := s.getFromCache(cacheKey); cached != nil {
		s.logger.Debug("Returning cached search results")
		return cached, nil
	}

	// Get all available tools from database
	tools, err := s.db.GetAllTools()
	if err != nil {
		return nil, fmt.Errorf("failed to query tools: %w", err)
	}

	if len(tools) == 0 {
		s.logger.Warn("No tools available in database")
		return []SearchResult{}, nil
	}

	s.logger.Debug("Retrieved tools from database", zap.Int("count", len(tools)))

	// Try LLM sampling first
	results, err := s.searchWithSampling(ctx, taskDescription, tools, maxResults)
	if err != nil {
		s.logger.Warn("LLM sampling failed, falling back to keyword search", zap.Error(err))
		// Fallback to keyword-based search
		results = s.keywordSearch(taskDescription, tools, maxResults)
	}

	// Cache results
	s.putInCache(cacheKey, results)

	return results, nil
}

// searchWithSampling uses LLM to intelligently select tools
func (s *SearchService) searchWithSampling(ctx context.Context, taskDescription string, tools []database.ToolRecord, maxResults int) ([]SearchResult, error) {
	// Build tool catalog for the prompt
	toolCatalog := s.buildToolCatalog(tools)

	// Create system prompt for tool selection
	systemPrompt := `You are an expert tool selector. Given a task description and a catalog of available tools, you must select the most appropriate tools that can help accomplish the task.

Analyze the task description and the tool catalog carefully. Return a JSON array of tool recommendations, where each recommendation has:
- "name": the exact tool name from the catalog
- "server": the server name the tool belongs to
- "score": a relevance score from 0.0 to 1.0 (1.0 being most relevant)
- "reason": brief explanation of why this tool is relevant

Return ONLY the JSON array, no other text. If no tools are relevant, return an empty array [].`

	userPrompt := fmt.Sprintf(`Task: %s

Available Tools:
%s

Select the top %d most relevant tools for this task.`, taskDescription, toolCatalog, maxResults)

	// Call LLM based on provider
	var responseText string
	var err error

	switch strings.ToLower(s.samplingCfg.Provider) {
	case "anthropic":
		responseText, err = s.callAnthropic(ctx, systemPrompt, userPrompt)
	case "openai":
		responseText, err = s.callOpenAI(ctx, systemPrompt, userPrompt)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", s.samplingCfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse LLM response
	results, err := s.parseSamplingResponse(responseText, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	s.logger.Info("LLM sampling completed", zap.Int("results", len(results)))
	return results, nil
}

// buildToolCatalog creates a formatted catalog of tools for the prompt
func (s *SearchService) buildToolCatalog(tools []database.ToolRecord) string {
	var sb strings.Builder

	for i, tool := range tools {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("- [%s] %s: %s", tool.ServerID, tool.Name, tool.Description))
	}

	return sb.String()
}

// parseSamplingResponse parses the LLM response and creates search results
func (s *SearchService) parseSamplingResponse(responseText string, tools []database.ToolRecord) ([]SearchResult, error) {
	// Extract JSON from response (LLMs sometimes add text before/after)
	jsonStart := strings.Index(responseText, "[")
	jsonEnd := strings.LastIndex(responseText, "]")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonText := responseText[jsonStart : jsonEnd+1]

	// Parse JSON array
	var recommendations []struct {
		Name   string  `json:"name"`
		Server string  `json:"server"`
		Score  float64 `json:"score"`
		Reason string  `json:"reason"`
	}

	if err := json.Unmarshal([]byte(jsonText), &recommendations); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Create search results by matching with actual tools
	var results []SearchResult

	for _, rec := range recommendations {
		// Find the tool in the database
		for _, tool := range tools {
			if tool.Name == rec.Name && tool.ServerID == rec.Server {
				// Parse input schema
				var schema map[string]interface{}
				if err := json.Unmarshal([]byte(tool.InputSchema), &schema); err != nil {
					s.logger.Warn("Failed to parse input schema",
						zap.String("tool", tool.Name),
						zap.Error(err),
					)
					schema = make(map[string]interface{})
				}

				results = append(results, SearchResult{
					Name:        tool.Name,
					Server:      tool.ServerID,
					Description: tool.Description,
					InputSchema: schema,
					Score:       rec.Score,
				})
				break
			}
		}
	}

	return results, nil
}

// keywordSearch performs simple keyword-based search as fallback
func (s *SearchService) keywordSearch(query string, tools []database.ToolRecord, maxResults int) []SearchResult {
	s.logger.Debug("Performing keyword-based search")

	queryLower := strings.ToLower(query)
	keywords := strings.Fields(queryLower)

	// Score each tool based on keyword matches
	type scoredTool struct {
		tool  database.ToolRecord
		score float64
	}

	var scoredTools []scoredTool

	for _, tool := range tools {
		score := 0.0
		toolText := strings.ToLower(tool.Name + " " + tool.Description)

		// Count keyword matches
		for _, keyword := range keywords {
			if strings.Contains(toolText, keyword) {
				score += 1.0
			}
		}

		// Bonus for name match
		if strings.Contains(strings.ToLower(tool.Name), queryLower) {
			score += 5.0
		}

		if score > 0 {
			scoredTools = append(scoredTools, scoredTool{tool: tool, score: score})
		}
	}

	// Sort by score (simple bubble sort for small lists)
	for i := 0; i < len(scoredTools); i++ {
		for j := i + 1; j < len(scoredTools); j++ {
			if scoredTools[j].score > scoredTools[i].score {
				scoredTools[i], scoredTools[j] = scoredTools[j], scoredTools[i]
			}
		}
	}

	// Convert to results
	var results []SearchResult
	for i := 0; i < len(scoredTools) && i < maxResults; i++ {
		tool := scoredTools[i].tool

		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(tool.InputSchema), &schema); err != nil {
			schema = make(map[string]interface{})
		}

		results = append(results, SearchResult{
			Name:        tool.Name,
			Server:      tool.ServerID,
			Description: tool.Description,
			InputSchema: schema,
			Score:       scoredTools[i].score / 10.0, // Normalize score
		})
	}

	s.logger.Info("Keyword search completed", zap.Int("results", len(results)))
	return results
}

// getFromCache retrieves cached search results if not expired
func (s *SearchService) getFromCache(key string) []SearchResult {
	s.cacheMux.RLock()
	defer s.cacheMux.RUnlock()

	cached, exists := s.cache[key]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(cached.Timestamp) > s.cacheTTL {
		return nil
	}

	return cached.Results
}

// putInCache stores search results in cache
func (s *SearchService) putInCache(key string, results []SearchResult) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()

	s.cache[key] = &cachedSearch{
		Results:   results,
		Timestamp: time.Now(),
	}

	// Clean old cache entries (simple cleanup)
	if len(s.cache) > 100 {
		for k, v := range s.cache {
			if time.Since(v.Timestamp) > s.cacheTTL {
				delete(s.cache, k)
			}
		}
	}
}
