package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// MedicalDisclaimer manages medical disclaimers for all outputs
type MedicalDisclaimer struct {
	mu              sync.RWMutex
	disclaimers     map[string]DisclaimerConfig
	defaultLanguage string
	enabled         bool
	auditLog        []DisclaimerAudit
	maxAuditEntries int
}

// DisclaimerConfig represents disclaimer configuration for different contexts
type DisclaimerConfig struct {
	ID          string            `json:"id"`
	Context     string            `json:"context"`   // "nutrition", "recipe", "health", "general"
	Languages   map[string]string `json:"languages"` // language code -> disclaimer text
	Severity    string            `json:"severity"`  // "critical", "warning", "info"
	Placement   string            `json:"placement"` // "header", "footer", "inline"
	Required    bool              `json:"required"`  // must be shown
	Formatting  FormatConfig      `json:"formatting"`
	Triggers    []string          `json:"triggers"`   // keywords that trigger this disclaimer
	Exclusions  []string          `json:"exclusions"` // contexts where this shouldn't appear
	LastUpdated time.Time         `json:"last_updated"`
}

// FormatConfig defines how disclaimers should be formatted
type FormatConfig struct {
	Bold       bool   `json:"bold"`
	Italic     bool   `json:"italic"`
	Color      string `json:"color"`
	Background string `json:"background"`
	Border     bool   `json:"border"`
	Icon       string `json:"icon"`
	FontSize   string `json:"font_size"`
	Margin     string `json:"margin"`
	Padding    string `json:"padding"`
}

// DisclaimerAudit represents an audit log entry
type DisclaimerAudit struct {
	Timestamp   time.Time `json:"timestamp"`
	UserID      string    `json:"user_id,omitempty"`
	Context     string    `json:"context"`
	Disclaimers []string  `json:"disclaimers"`
	Content     string    `json:"content"`
	Language    string    `json:"language"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
}

// DisclaimerResponse represents the response with embedded disclaimers
type DisclaimerResponse struct {
	Content     interface{}          `json:"content"`
	Disclaimers []EmbeddedDisclaimer `json:"disclaimers"`
	Metadata    DisclaimerMetadata   `json:"metadata"`
}

// EmbeddedDisclaimer represents a disclaimer embedded in content
type EmbeddedDisclaimer struct {
	ID        string       `json:"id"`
	Text      string       `json:"text"`
	Context   string       `json:"context"`
	Severity  string       `json:"severity"`
	Placement string       `json:"placement"`
	Format    FormatConfig `json:"format"`
	Language  string       `json:"language"`
	Required  bool         `json:"required"`
}

// DisclaimerMetadata provides metadata about disclaimer application
type DisclaimerMetadata struct {
	AppliedCount    int       `json:"applied_count"`
	Language        string    `json:"language"`
	Context         string    `json:"context"`
	Timestamp       time.Time `json:"timestamp"`
	ComplianceLevel string    `json:"compliance_level"`
}

// NewMedicalDisclaimer creates a new medical disclaimer service
func NewMedicalDisclaimer() *MedicalDisclaimer {
	md := &MedicalDisclaimer{
		disclaimers:     make(map[string]DisclaimerConfig),
		defaultLanguage: "en",
		enabled:         true,
		auditLog:        make([]DisclaimerAudit, 0),
		maxAuditEntries: 10000,
	}

	// Initialize default disclaimers
	md.initializeDefaultDisclaimers()

	return md
}

// initializeDefaultDisclaimers sets up default medical disclaimers
func (md *MedicalDisclaimer) initializeDefaultDisclaimers() {
	// Critical medical disclaimer
	md.disclaimers["medical_critical"] = DisclaimerConfig{
		ID:      "medical_critical",
		Context: "health",
		Languages: map[string]string{
			"en": "⚠️ MEDICAL DISCLAIMER: This information is for educational purposes only and is not intended as medical advice. Always consult with a qualified healthcare professional before making any dietary changes, especially if you have medical conditions, allergies, or are taking medications. Individual nutritional needs vary significantly.",
			"ar": "⚠️ إخلاء مسؤولية طبية: هذه المعلومات لأغراض تعليمية فقط وليست المقصود بها كنصيحة طبية. استشر دائماً أخصائي رعاية صحية مؤهل قبل إجراء أي تغييرات غذائية، خاصة إذا كان لديك حالات طبية أو حساسية أو تتناول أدوية. الاحتياجات الغذائية الفردية تختلف بشكل كبير.",
		},
		Severity:  "critical",
		Placement: "header",
		Required:  true,
		Formatting: FormatConfig{
			Bold:       true,
			Color:      "#d32f2f",
			Background: "#ffebee",
			Border:     true,
			Icon:       "⚠️",
			FontSize:   "14px",
			Margin:     "10px 0",
			Padding:    "12px",
		},
		Triggers:    []string{"health", "medical", "diet", "nutrition", "calories", "weight", "diabetes", "allergy"},
		LastUpdated: time.Now(),
	}

	// Nutrition disclaimer
	md.disclaimers["nutrition_general"] = DisclaimerConfig{
		ID:      "nutrition_general",
		Context: "nutrition",
		Languages: map[string]string{
			"en": "📊 NUTRITION NOTICE: Nutritional values are estimates based on standard food databases and may vary depending on preparation methods, ingredient brands, and portion sizes. For precise nutritional information, consult product labels or a registered dietitian.",
			"ar": "📊 ملاحظة غذائية: القيم الغذائية تقديرية بناءً على قواعد بيانات الطعام المعيارية وقد تختلف حسب طرق التحضير وعلامات المكونات وأحجام الحصص. للحصول على معلومات غذائية دقيقة، استشر ملصقات المنتجات أو أخصائي تغذية مسجل.",
		},
		Severity:  "warning",
		Placement: "footer",
		Required:  true,
		Formatting: FormatConfig{
			Bold:       false,
			Italic:     true,
			Color:      "#f57c00",
			Background: "#fff8e1",
			Border:     false,
			Icon:       "📊",
			FontSize:   "12px",
			Margin:     "8px 0",
			Padding:    "8px",
		},
		Triggers:    []string{"calories", "protein", "carbs", "fat", "vitamins", "minerals", "nutrition"},
		LastUpdated: time.Now(),
	}

	// Recipe disclaimer
	md.disclaimers["recipe_safety"] = DisclaimerConfig{
		ID:      "recipe_safety",
		Context: "recipe",
		Languages: map[string]string{
			"en": "👨‍🍳 RECIPE SAFETY: Always follow proper food safety guidelines. Cook meats to safe internal temperatures, wash produce thoroughly, and be aware of cross-contamination risks. If you have food allergies, carefully review all ingredients.",
			"ar": "👨‍🍳 سلامة الوصفات: اتبع دائماً إرشادات سلامة الطعام المناسبة. اطبخ اللحوم إلى درجات حرارة داخلية آمنة، واغسل المنتجات جيداً، وكن على علم بمخاطر التلوث المتبادل. إذا كان لديك حساسية طعام، راجع جميع المكونات بعناية.",
		},
		Severity:  "warning",
		Placement: "inline",
		Required:  false,
		Formatting: FormatConfig{
			Bold:       false,
			Color:      "#388e3c",
			Background: "#e8f5e8",
			Border:     false,
			Icon:       "👨‍🍳",
			FontSize:   "12px",
			Margin:     "5px 0",
			Padding:    "6px",
		},
		Triggers:    []string{"recipe", "cooking", "ingredients", "preparation"},
		LastUpdated: time.Now(),
	}

	// Allergy disclaimer
	md.disclaimers["allergy_warning"] = DisclaimerConfig{
		ID:      "allergy_warning",
		Context: "allergy",
		Languages: map[string]string{
			"en": "🚨 ALLERGY WARNING: This platform cannot guarantee the absence of allergens. Always check ingredient labels and consult with healthcare providers if you have severe allergies. Cross-contamination may occur during food preparation.",
			"ar": "🚨 تحذير من الحساسية: لا يمكن لهذه المنصة ضمان عدم وجود مسببات الحساسية. تحقق دائماً من ملصقات المكونات واستشر مقدمي الرعاية الصحية إذا كان لديك حساسية شديدة. قد يحدث تلوث متبادل أثناء تحضير الطعام.",
		},
		Severity:  "critical",
		Placement: "header",
		Required:  true,
		Formatting: FormatConfig{
			Bold:       true,
			Color:      "#d32f2f",
			Background: "#ffcdd2",
			Border:     true,
			Icon:       "🚨",
			FontSize:   "13px",
			Margin:     "8px 0",
			Padding:    "10px",
		},
		Triggers:    []string{"allergy", "allergen", "nuts", "dairy", "gluten", "shellfish", "eggs"},
		LastUpdated: time.Now(),
	}

	// Weight management disclaimer
	md.disclaimers["weight_management"] = DisclaimerConfig{
		ID:      "weight_management",
		Context: "weight",
		Languages: map[string]string{
			"en": "⚖️ WEIGHT MANAGEMENT: Weight loss/gain recommendations are general guidelines only. Sustainable weight management requires personalized approaches. Consult healthcare professionals for safe and effective weight management strategies.",
			"ar": "⚖️ إدارة الوزن: توصيات فقدان/زيادة الوزن هي إرشادات عامة فقط. إدارة الوزن المستدامة تتطلب نهج شخصية. استشر المهنيين الصحيين للحصول على استراتيجيات إدارة الوزن الآمنة والفعالة.",
		},
		Severity:  "warning",
		Placement: "footer",
		Required:  true,
		Formatting: FormatConfig{
			Bold:       false,
			Italic:     true,
			Color:      "#7b1fa2",
			Background: "#f3e5f5",
			Border:     false,
			Icon:       "⚖️",
			FontSize:   "12px",
			Margin:     "6px 0",
			Padding:    "8px",
		},
		Triggers:    []string{"weight", "lose", "gain", "bmi", "obesity", "diet plan"},
		LastUpdated: time.Now(),
	}
}

// EmbedDisclaimers embeds appropriate disclaimers into content
func (md *MedicalDisclaimer) EmbedDisclaimers(content interface{}, context string, language string, userID string, ipAddress string, userAgent string) (*DisclaimerResponse, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	if !md.enabled {
		return &DisclaimerResponse{
			Content:     content,
			Disclaimers: []EmbeddedDisclaimer{},
			Metadata: DisclaimerMetadata{
				AppliedCount:    0,
				Language:        language,
				Context:         context,
				Timestamp:       time.Now(),
				ComplianceLevel: "disabled",
			},
		}, nil
	}

	if language == "" {
		language = md.defaultLanguage
	}

	// Convert content to string for trigger analysis
	contentStr := md.contentToString(content)

	// Find applicable disclaimers
	applicableDisclaimers := md.findApplicableDisclaimers(contentStr, context, language)

	// Create embedded disclaimers
	embeddedDisclaimers := make([]EmbeddedDisclaimer, 0, len(applicableDisclaimers))
	disclaimerIDs := make([]string, 0, len(applicableDisclaimers))

	for _, disclaimer := range applicableDisclaimers {
		text, exists := disclaimer.Languages[language]
		if !exists {
			// Fallback to default language
			text, exists = disclaimer.Languages[md.defaultLanguage]
			if !exists {
				continue
			}
		}

		embedded := EmbeddedDisclaimer{
			ID:        disclaimer.ID,
			Text:      text,
			Context:   disclaimer.Context,
			Severity:  disclaimer.Severity,
			Placement: disclaimer.Placement,
			Format:    disclaimer.Formatting,
			Language:  language,
			Required:  disclaimer.Required,
		}

		embeddedDisclaimers = append(embeddedDisclaimers, embedded)
		disclaimerIDs = append(disclaimerIDs, disclaimer.ID)
	}

	// Log disclaimer usage
	md.logDisclaimerUsage(userID, context, disclaimerIDs, contentStr, language, ipAddress, userAgent)

	// Create response
	response := &DisclaimerResponse{
		Content:     content,
		Disclaimers: embeddedDisclaimers,
		Metadata: DisclaimerMetadata{
			AppliedCount:    len(embeddedDisclaimers),
			Language:        language,
			Context:         context,
			Timestamp:       time.Now(),
			ComplianceLevel: md.getComplianceLevel(embeddedDisclaimers),
		},
	}

	return response, nil
}

// findApplicableDisclaimers finds disclaimers that should be applied
func (md *MedicalDisclaimer) findApplicableDisclaimers(content, context, language string) []DisclaimerConfig {
	applicable := make([]DisclaimerConfig, 0)
	contentLower := strings.ToLower(content)
	contextLower := strings.ToLower(context)

	for _, disclaimer := range md.disclaimers {
		// Check if excluded from this context
		excluded := false
		for _, exclusion := range disclaimer.Exclusions {
			if strings.Contains(contextLower, strings.ToLower(exclusion)) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check context match
		contextMatch := disclaimer.Context == "general" ||
			strings.Contains(contextLower, strings.ToLower(disclaimer.Context))

		// Check trigger words
		triggerMatch := len(disclaimer.Triggers) == 0 // If no triggers, always match
		for _, trigger := range disclaimer.Triggers {
			if strings.Contains(contentLower, strings.ToLower(trigger)) {
				triggerMatch = true
				break
			}
		}

		// Check if language is supported
		languageSupported := false
		for lang := range disclaimer.Languages {
			if lang == language || lang == md.defaultLanguage {
				languageSupported = true
				break
			}
		}

		if contextMatch && triggerMatch && languageSupported {
			applicable = append(applicable, disclaimer)
		}
	}

	return applicable
}

// contentToString converts content to string for analysis
func (md *MedicalDisclaimer) contentToString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// Try to marshal to JSON and extract text
		if jsonBytes, err := json.Marshal(content); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", content)
	}
}

// logDisclaimerUsage logs disclaimer usage for audit purposes
func (md *MedicalDisclaimer) logDisclaimerUsage(userID, context string, disclaimerIDs []string, content, language, ipAddress, userAgent string) {
	auditEntry := DisclaimerAudit{
		Timestamp:   time.Now(),
		UserID:      userID,
		Context:     context,
		Disclaimers: disclaimerIDs,
		Content:     md.truncateContent(content, 500), // Limit content length
		Language:    language,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	// Add to audit log (thread-safe)
	go func() {
		md.mu.Lock()
		defer md.mu.Unlock()

		md.auditLog = append(md.auditLog, auditEntry)

		// Trim audit log if it exceeds max entries
		if len(md.auditLog) > md.maxAuditEntries {
			md.auditLog = md.auditLog[len(md.auditLog)-md.maxAuditEntries:]
		}
	}()
}

// truncateContent truncates content to specified length
func (md *MedicalDisclaimer) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

// getComplianceLevel determines compliance level based on applied disclaimers
func (md *MedicalDisclaimer) getComplianceLevel(disclaimers []EmbeddedDisclaimer) string {
	hasCritical := false
	hasWarning := false

	for _, disclaimer := range disclaimers {
		switch disclaimer.Severity {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		}
	}

	if hasCritical {
		return "high"
	} else if hasWarning {
		return "medium"
	}
	return "low"
}

// AddDisclaimer adds a new disclaimer configuration
func (md *MedicalDisclaimer) AddDisclaimer(disclaimer DisclaimerConfig) {
	md.mu.Lock()
	defer md.mu.Unlock()

	disclaimer.LastUpdated = time.Now()
	md.disclaimers[disclaimer.ID] = disclaimer
}

// UpdateDisclaimer updates an existing disclaimer
func (md *MedicalDisclaimer) UpdateDisclaimer(id string, disclaimer DisclaimerConfig) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if _, exists := md.disclaimers[id]; !exists {
		return fmt.Errorf("disclaimer with ID %s not found", id)
	}

	disclaimer.ID = id
	disclaimer.LastUpdated = time.Now()
	md.disclaimers[id] = disclaimer

	return nil
}

// RemoveDisclaimer removes a disclaimer
func (md *MedicalDisclaimer) RemoveDisclaimer(id string) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if _, exists := md.disclaimers[id]; !exists {
		return fmt.Errorf("disclaimer with ID %s not found", id)
	}

	delete(md.disclaimers, id)
	return nil
}

// GetDisclaimers returns all disclaimers
func (md *MedicalDisclaimer) GetDisclaimers() map[string]DisclaimerConfig {
	md.mu.RLock()
	defer md.mu.RUnlock()

	disclaimers := make(map[string]DisclaimerConfig)
	for k, v := range md.disclaimers {
		disclaimers[k] = v
	}

	return disclaimers
}

// GetAuditLog returns recent audit log entries
func (md *MedicalDisclaimer) GetAuditLog(limit int) []DisclaimerAudit {
	md.mu.RLock()
	defer md.mu.RUnlock()

	if limit <= 0 || limit > len(md.auditLog) {
		limit = len(md.auditLog)
	}

	// Return most recent entries
	start := len(md.auditLog) - limit
	if start < 0 {
		start = 0
	}

	return md.auditLog[start:]
}

// SetEnabled enables or disables disclaimer embedding
func (md *MedicalDisclaimer) SetEnabled(enabled bool) {
	md.mu.Lock()
	defer md.mu.Unlock()
	md.enabled = enabled
}

// IsEnabled returns whether disclaimer embedding is enabled
func (md *MedicalDisclaimer) IsEnabled() bool {
	md.mu.RLock()
	defer md.mu.RUnlock()
	return md.enabled
}

// RegisterRoutes registers medical disclaimer API routes
func (md *MedicalDisclaimer) RegisterRoutes(e *echo.Group) {
	e.POST("/disclaimers/embed", md.handleEmbedDisclaimers)
	e.GET("/disclaimers", md.handleGetDisclaimers)
	e.POST("/disclaimers", md.handleAddDisclaimer)
	e.PUT("/disclaimers/:id", md.handleUpdateDisclaimer)
	e.DELETE("/disclaimers/:id", md.handleRemoveDisclaimer)
	e.GET("/disclaimers/audit", md.handleGetAuditLog)
	e.PUT("/disclaimers/settings", md.handleUpdateSettings)
}

// API Handlers

type EmbedDisclaimersRequest struct {
	Content  interface{} `json:"content"`
	Context  string      `json:"context"`
	Language string      `json:"language,omitempty"`
	UserID   string      `json:"user_id,omitempty"`
}

type UpdateDisclaimerSettingsRequest struct {
	Enabled         *bool  `json:"enabled,omitempty"`
	DefaultLanguage string `json:"default_language,omitempty"`
}

func (md *MedicalDisclaimer) handleEmbedDisclaimers(c echo.Context) error {
	var req EmbedDisclaimersRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	response, err := md.EmbedDisclaimers(req.Content, req.Context, req.Language, req.UserID, ipAddress, userAgent)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, response)
}

func (md *MedicalDisclaimer) handleGetDisclaimers(c echo.Context) error {
	disclaimers := md.GetDisclaimers()
	return c.JSON(200, disclaimers)
}

func (md *MedicalDisclaimer) handleAddDisclaimer(c echo.Context) error {
	var disclaimer DisclaimerConfig
	if err := c.Bind(&disclaimer); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid disclaimer format"})
	}

	md.AddDisclaimer(disclaimer)
	return c.JSON(201, map[string]string{"message": "Disclaimer added successfully"})
}

func (md *MedicalDisclaimer) handleUpdateDisclaimer(c echo.Context) error {
	id := c.Param("id")
	var disclaimer DisclaimerConfig
	if err := c.Bind(&disclaimer); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid disclaimer format"})
	}

	if err := md.UpdateDisclaimer(id, disclaimer); err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Disclaimer updated successfully"})
}

func (md *MedicalDisclaimer) handleRemoveDisclaimer(c echo.Context) error {
	id := c.Param("id")
	if err := md.RemoveDisclaimer(id); err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Disclaimer removed successfully"})
}

func (md *MedicalDisclaimer) handleGetAuditLog(c echo.Context) error {
	limit := 100 // Default limit
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || parsedLimit != 1 {
			limit = 100
		}
	}

	auditLog := md.GetAuditLog(limit)
	return c.JSON(200, map[string]interface{}{
		"audit_log": auditLog,
		"count":     len(auditLog),
	})
}

func (md *MedicalDisclaimer) handleUpdateSettings(c echo.Context) error {
	var req UpdateDisclaimerSettingsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	md.mu.Lock()
	defer md.mu.Unlock()

	if req.Enabled != nil {
		md.enabled = *req.Enabled
	}

	if req.DefaultLanguage != "" {
		md.defaultLanguage = req.DefaultLanguage
	}

	return c.JSON(200, map[string]string{"message": "Settings updated successfully"})
}
