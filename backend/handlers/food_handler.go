package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"nutrition-platform/models"
	"nutrition-platform/repositories"

	"github.com/labstack/echo/v4"
)

// FoodHandler handles food CRUD operations
type FoodHandler struct {
	foodRepo *repositories.FoodRepository
}

// NewFoodHandler creates a new food handler
func NewFoodHandler(db *sql.DB) *FoodHandler {
	return &FoodHandler{
		foodRepo: repositories.NewFoodRepository(db),
	}
}

// GetFoods returns paginated list of foods
func (h *FoodHandler) GetFoods(c echo.Context) error {
	// Get user ID from context
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to string (repository expects string)
	var userIDStr string
	switch v := userID.(type) {
	case uint:
		userIDStr = strconv.FormatUint(uint64(v), 10)
	case string:
		userIDStr = v
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Parse pagination parameters
	page := 1
	limit := 20
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Parse search query
	searchQuery := c.QueryParam("search")

	// Parse filters
	filters := models.FoodSearchFilters{
		Brand:       c.QueryParam("brand"),
		SourceType:  c.QueryParam("source_type"),
		SortBy:      c.QueryParam("sort_by"),
		SortDirection: c.QueryParam("sort_direction"),
	}

	if verifiedStr := c.QueryParam("verified"); verifiedStr != "" {
		verified := verifiedStr == "true"
		filters.Verified = &verified
	}

	// Get foods
	offset := (page - 1) * limit
	foods, err := h.foodRepo.SearchFoods(userIDStr, searchQuery, filters, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch foods: " + err.Error(),
		})
	}

	// TODO: Get total count for pagination metadata
	total := len(foods) // Placeholder - repository should return total count

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   foods,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_pages": (total + limit - 1) / limit,
			"has_next":   page*limit < total,
			"has_prev":   page > 1,
		},
	})
}

// GetFood returns a specific food by ID
func (h *FoodHandler) GetFood(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case uint:
		userIDStr = strconv.FormatUint(uint64(v), 10)
	case string:
		userIDStr = v
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	foodID := c.Param("id")
	food, err := h.foodRepo.GetFoodByID(foodID, userIDStr)
	if err != nil {
		if err.Error() == "food not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Food not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch food: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   food,
	})
}

// CreateFood creates a new food
func (h *FoodHandler) CreateFood(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to uint pointer
	var userIDUint *uint
	switch v := userID.(type) {
	case uint:
		userIDUint = &v
	case string:
		if id, err := strconv.ParseUint(v, 10, 32); err == nil {
			uid := uint(id)
			userIDUint = &uid
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid user ID",
			})
		}
	case int:
		uid := uint(v)
		userIDUint = &uid
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	var req models.CreateFoodRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Convert to Food model - repository expects BarCode, Verified, SourceType
	food := &models.Food{
		UserID:       userIDUint,
		Name:         req.Name,
		Description:  req.Description,
		Brand:        req.Brand,
		Barcode:      req.Barcode,
		BarCode:      req.Barcode, // Repository uses BarCode field
		Category:     req.Category,
		Calories:     req.Calories,
		Protein:      req.Protein,
		Carbs:        req.Carbs,
		Fat:          req.Fat,
		SaturatedFat: 0, // Default value
		Fiber:        req.Fiber,
		Sugar:        req.Sugar,
		Sodium:       req.Sodium,
		Cholesterol:  0, // Default value
		Potassium:    0, // Default value
		ServingSize:  req.ServingSize,
		ServingUnit:  req.ServingUnit,
		SourceType:   "user",
		IsVerified:   false, // User-created foods are not verified by default
		Verified:     false, // Repository uses Verified field (not IsVerified)
	}

	// Note: Repository expects different structure - need to check actual repository model
	// For now, this is a placeholder that needs adjustment based on repository expectations
	err := h.foodRepo.CreateFood(food)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create food: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Food created successfully",
		"data":    food,
	})
}

// UpdateFood updates an existing food
func (h *FoodHandler) UpdateFood(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case uint:
		userIDStr = strconv.FormatUint(uint64(v), 10)
	case string:
		userIDStr = v
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	foodID := c.Param("id")

	// Get existing food
	existingFood, err := h.foodRepo.GetFoodByID(foodID, userIDStr)
	if err != nil {
		if err.Error() == "food not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Food not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch food: " + err.Error(),
		})
	}

	// Bind update request
	var req models.UpdateFoodRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Update fields
	if req.Name != nil {
		existingFood.Name = *req.Name
	}
	if req.Description != nil {
		existingFood.Description = req.Description
	}
	if req.Brand != nil {
		existingFood.Brand = req.Brand
	}
	if req.Barcode != nil {
		existingFood.Barcode = req.Barcode
	}
	if req.Category != nil {
		existingFood.Category = req.Category
	}
	if req.Calories != nil {
		existingFood.Calories = *req.Calories
	}
	if req.Protein != nil {
		existingFood.Protein = *req.Protein
	}
	if req.Carbs != nil {
		existingFood.Carbs = *req.Carbs
	}
	if req.Fat != nil {
		existingFood.Fat = *req.Fat
	}
	if req.Fiber != nil {
		existingFood.Fiber = *req.Fiber
	}
	if req.Sugar != nil {
		existingFood.Sugar = *req.Sugar
	}
	if req.Sodium != nil {
		existingFood.Sodium = *req.Sodium
	}
	if req.ServingSize != nil {
		existingFood.ServingSize = *req.ServingSize
	}
	if req.ServingUnit != nil {
		existingFood.ServingUnit = *req.ServingUnit
	}

	err = h.foodRepo.UpdateFood(existingFood)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update food: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Food updated successfully",
		"data":    existingFood,
	})
}

// DeleteFood deletes a food
func (h *FoodHandler) DeleteFood(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case uint:
		userIDStr = strconv.FormatUint(uint64(v), 10)
	case string:
		userIDStr = v
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	foodID := c.Param("id")
	err := h.foodRepo.DeleteFood(foodID, userIDStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete food: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Food deleted successfully",
	})
}

// SearchFoods searches for foods
func (h *FoodHandler) SearchFoods(c echo.Context) error {
	return h.GetFoods(c) // Reuse GetFoods which already supports search
}

