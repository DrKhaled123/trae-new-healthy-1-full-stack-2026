package utils

import (
	"fmt"
	"strings"
)

// SearchQuery represents a search query
type SearchQuery struct {
	Query      string
	Fields     []string // Fields to search in
	Language   string   // "en", "ar", or "both"
	FuzzyMatch bool     // Enable fuzzy matching
}

// BuildSearchClause builds a SQL LIKE clause for searching
func BuildSearchClause(query SearchQuery, tablePrefix string) (string, []interface{}) {
	if query.Query == "" {
		return "", nil
	}

	// Sanitize query - remove SQL injection risks
	query.Query = strings.TrimSpace(query.Query)
	query.Query = strings.ReplaceAll(query.Query, "%", "\\%")
	query.Query = strings.ReplaceAll(query.Query, "_", "\\_")

	searchTerm := "%" + query.Query + "%"
	var conditions []string
	var args []interface{}

	// If no fields specified, use default fields
	if len(query.Fields) == 0 {
		query.Fields = []string{"name", "description", "content"}
	}

	// Build conditions for each field
	for _, field := range query.Fields {
		fieldName := field
		if tablePrefix != "" {
			fieldName = tablePrefix + "." + field
		}

		// Support bilingual search
		if query.Language == "both" || query.Language == "" {
			// Search in both English and Arabic fields
			conditions = append(conditions, fmt.Sprintf("(%s LIKE ? OR %s_ar LIKE ?)", fieldName, fieldName))
			args = append(args, searchTerm, searchTerm)
		} else if query.Language == "ar" {
			// Search only in Arabic field
			conditions = append(conditions, fmt.Sprintf("%s_ar LIKE ?", fieldName))
			args = append(args, searchTerm)
		} else {
			// Search only in English field (default)
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", fieldName))
			args = append(args, searchTerm)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	clause := "(" + strings.Join(conditions, " OR ") + ")"
	return clause, args
}

// CalculateRelevanceScore calculates a relevance score for search results
func CalculateRelevanceScore(text, query string) int {
	text = strings.ToLower(text)
	query = strings.ToLower(query)

	score := 0

	// Exact match gets highest score
	if strings.Contains(text, query) {
		score += 100
	}

	// Check for word matches
	queryWords := strings.Fields(query)
	for _, word := range queryWords {
		if strings.Contains(text, word) {
			score += 10
		}
	}

	// Check for partial word matches
	for _, word := range queryWords {
		if len(word) > 3 {
			for i := 0; i < len(word)-2; i++ {
				substring := word[i : i+3]
				if strings.Contains(text, substring) {
					score += 1
					break
				}
			}
		}
	}

	return score
}

// SanitizeSearchQuery sanitizes a search query to prevent SQL injection
func SanitizeSearchQuery(query string) string {
	// Remove potentially dangerous characters
	query = strings.TrimSpace(query)
	query = strings.ReplaceAll(query, "%", "")
	query = strings.ReplaceAll(query, "_", "")
	query = strings.ReplaceAll(query, "'", "")
	query = strings.ReplaceAll(query, "\"", "")
	query = strings.ReplaceAll(query, ";", "")
	query = strings.ReplaceAll(query, "--", "")
	query = strings.ReplaceAll(query, "/*", "")
	query = strings.ReplaceAll(query, "*/", "")
	return query
}

