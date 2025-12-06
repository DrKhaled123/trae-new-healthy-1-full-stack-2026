package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nutrition-platform/utils"
	"github.com/labstack/echo/v4"
)

type InjuryHandler struct {
	dataDir string
}

func NewInjuryHandler(dataDir string) *InjuryHandler {
	return &InjuryHandler{dataDir: dataDir}
}

// GetInjuries returns a list of all available injuries
func (h *InjuryHandler) GetInjuries(c echo.Context) error {
	injuriesDir := filepath.Join(h.dataDir, "../injury easy trae json")

	files, err := os.ReadDir(injuriesDir)
	if err != nil {
		return utils.InternalServerErrorResponse(c, "Failed to read injuries directory")
	}

	injuries := []map[string]interface{}{}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".js" {
			data, err := os.ReadFile(filepath.Join(injuriesDir, file.Name()))
			if err != nil {
				continue // Skip files that can't be read
			}

			// Parse the file to extract the title
			content := string(data)
			title := h.extractTitle(content, file.Name())

			injuries = append(injuries, map[string]interface{}{
				"id":    strings.TrimSuffix(file.Name(), ".js"),
				"title": title,
				"file":  file.Name(),
			})
		}
	}

	// Parse pagination parameters
	params, _ := utils.ParsePagination(c)
	paginationMeta := utils.CalculatePagination(params.Page, params.Limit, len(injuries))
	// Convert PaginationMeta to *Pagination
	pagination := &utils.Pagination{
		Page:       paginationMeta.Page,
		Limit:      paginationMeta.Limit,
		Total:      paginationMeta.Total,
		TotalPages: paginationMeta.TotalPages,
		HasNext:    paginationMeta.HasNext,
		HasPrev:    paginationMeta.HasPrev,
	}
	return utils.SuccessListWithPagination(c, injuries, pagination)
}

// GetInjury returns a specific injury by ID
func (h *InjuryHandler) GetInjury(c echo.Context) error {
	injuryID := c.Param("id")
	filePath := filepath.Join(h.dataDir, "../injury easy trae json", injuryID+".js")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return utils.NotFoundResponse(c, "Injury not found")
	}

	// Parse the JSON content from the file
	content := string(data)
	injuryData := h.parseInjuryFile(content)

	return utils.SuccessResponse(c, injuryData)
}

// SearchInjuries allows searching across injury data
func (h *InjuryHandler) SearchInjuries(c echo.Context) error {
	query := strings.ToLower(c.QueryParam("q"))
	if query == "" {
		return h.GetInjuries(c)
	}

	injuriesDir := filepath.Join(h.dataDir, "../injury easy trae json")

	files, err := os.ReadDir(injuriesDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read injuries directory",
		})
	}

	results := []map[string]interface{}{}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".js" {
			data, err := os.ReadFile(filepath.Join(injuriesDir, file.Name()))
			if err != nil {
				continue
			}

			content := string(data)
			if strings.Contains(strings.ToLower(content), query) {
				title := h.extractTitle(content, file.Name())
				results = append(results, map[string]interface{}{
					"id":    strings.TrimSuffix(file.Name(), ".js"),
					"title": title,
					"file":  file.Name(),
				})
			}
		}
	}

	// Handle pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 10
	}

	start := (page - 1) * limit
	end := start + limit

	if start >= len(results) {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "success",
			"data":   []map[string]interface{}{},
			"total":  len(results),
			"page":   page,
			"limit":  limit,
		})
	}

	if end > len(results) {
		end = len(results)
	}

	paginatedResults := results[start:end]

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   paginatedResults,
		"total":  len(results),
		"page":   page,
		"limit":  limit,
	})
}

// GetInjuryCategories returns categorized injuries
func (h *InjuryHandler) GetInjuryCategories(c echo.Context) error {
	injuriesDir := filepath.Join(h.dataDir, "../injury easy trae json")

	files, err := os.ReadDir(injuriesDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read injuries directory",
		})
	}

	categories := map[string][]map[string]interface{}{}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".js" {
			data, err := os.ReadFile(filepath.Join(injuriesDir, file.Name()))
			if err != nil {
				continue
			}

			content := string(data)
			title := h.extractTitle(content, file.Name())
			category := h.categorizeInjury(file.Name(), content)

			injury := map[string]interface{}{
				"id":    strings.TrimSuffix(file.Name(), ".js"),
				"title": title,
				"file":  file.Name(),
			}

			categories[category] = append(categories[category], injury)
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "success",
		"categories": categories,
		"total":      len(files),
	})
}

// Helper functions

func (h *InjuryHandler) extractTitle(content, filename string) map[string]string {
	// Try to extract title from JSON content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"title": {`) {
			// Extract the title object
			startIndex := strings.Index(content, line)
			if startIndex == -1 {
				break
			}

			// Find the end of the title object
			braceCount := 0
			endIndex := startIndex
			for i, char := range content[startIndex:] {
				if char == '{' {
					braceCount++
				} else if char == '}' {
					braceCount--
					if braceCount == 0 {
						endIndex = startIndex + i + 1
						break
					}
				}
			}

			titleStr := content[startIndex:endIndex]
			var title map[string]string
			if err := json.Unmarshal([]byte(titleStr), &title); err == nil {
				return title
			}
			break
		}
	}

	// Fallback to filename-based title
	baseName := strings.TrimSuffix(filename, ".js")
	parts := strings.Split(baseName, " ")
	if len(parts) > 1 {
		baseName = strings.Join(parts[1:], " ")
	}

	return map[string]string{
		"english": baseName,
		"arabic":  baseName,
	}
}

func (h *InjuryHandler) categorizeInjury(filename, content string) string {
	filename = strings.ToLower(filename)
	content = strings.ToLower(content)

	// Categorize based on filename patterns and content
	if strings.Contains(filename, "neck") || strings.Contains(content, "neck") {
		return "Neck Injuries"
	}
	if strings.Contains(filename, "back") || strings.Contains(content, "back") {
		return "Back Injuries"
	}
	if strings.Contains(filename, "wrist") || strings.Contains(content, "wrist") {
		return "Wrist Injuries"
	}
	if strings.Contains(filename, "shoulder") || strings.Contains(content, "shoulder") {
		return "Shoulder Injuries"
	}
	if strings.Contains(filename, "knee") || strings.Contains(content, "knee") {
		return "Knee Injuries"
	}
	if strings.Contains(filename, "ankle") || strings.Contains(content, "ankle") {
		return "Ankle Injuries"
	}
	if strings.Contains(filename, "hip") || strings.Contains(content, "hip") {
		return "Hip Injuries"
	}

	return "Other Injuries"
}

func (h *InjuryHandler) parseInjuryFile(content string) map[string]interface{} {
	// Find JSON blocks in the file
	lines := strings.Split(content, "\n")
	inJsonBlock := false
	jsonLines := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "```json") {
			inJsonBlock = true
			continue
		}
		if strings.HasPrefix(line, "```") && inJsonBlock {
			inJsonBlock = false
			break
		}
		if inJsonBlock && line != "" {
			jsonLines = append(jsonLines, line)
		}
	}

	jsonStr := strings.Join(jsonLines, "\n")
	var injuryData map[string]interface{}

	if err := json.Unmarshal([]byte(jsonStr), &injuryData); err != nil {
		// If parsing fails, return a basic structure
		title := h.extractTitle(content, "unknown")
		return map[string]interface{}{
			"title":       title,
			"description": map[string]string{"english": "Failed to parse content", "arabic": "فشل في تحليل المحتوى"},
			"error":       "JSON parsing error",
		}
	}

	return injuryData
}
