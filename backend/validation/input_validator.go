package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"nutrition-platform/errors"

	"github.com/go-playground/validator/v10"
)

// InputValidator provides comprehensive input validation
type InputValidator struct {
	validator             *validator.Validate
	sqlInjectionPatterns  []string
	xssPatterns           []string
	commandPatterns       []string
	pathTraversalPatterns []string
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Field           string
	Required        bool
	MinLength       int
	MaxLength       int
	Pattern         string
	AllowedValues   []string
	CustomValidator func(value string) error
}

// ValidationContext provides context for validation
type ValidationContext struct {
	UserRole  string
	IPAddress string
	UserAgent string
	Endpoint  string
	Method    string
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	v := validator.New()
	
	// Register custom validations if needed
	
	return &InputValidator{
		validator:             v,
		sqlInjectionPatterns:  getSQLInjectionPatterns(),
		xssPatterns:           getXSSPatterns(),
		commandPatterns:       getCommandInjectionPatterns(),
		pathTraversalPatterns: getPathTraversalPatterns(),
	}
}

// Validate implements echo.Validator interface
func (v *InputValidator) Validate(i interface{}) error {
	// Use go-playground/validator for struct validation
	if v.validator == nil {
		return fmt.Errorf("validator not initialized")
	}
	
	err := v.validator.Struct(i)
	if err != nil {
		return err
	}
	
	return nil
}

// ValidateInput validates input against multiple security threats
func (iv *InputValidator) ValidateInput(input string, context *ValidationContext) error {
	// Check for SQL injection
	if err := iv.checkSQLInjection(input); err != nil {
		return err
	}

	// Check for XSS
	if err := iv.checkXSS(input); err != nil {
		return err
	}

	// Check for command injection
	if err := iv.checkCommandInjection(input); err != nil {
		return err
	}

	// Check for path traversal
	if err := iv.checkPathTraversal(input); err != nil {
		return err
	}

	// Check for suspicious patterns
	if err := iv.checkSuspiciousPatterns(input, context); err != nil {
		return err
	}

	return nil
}

// ValidateFields validates multiple fields with rules
func (iv *InputValidator) ValidateFields(data map[string]string, rules []ValidationRule, context *ValidationContext) *errors.ValidationErrors {
	validationErrors := errors.NewValidationErrors()

	for _, rule := range rules {
		value, exists := data[rule.Field]

		// Check required fields
		if rule.Required && (!exists || strings.TrimSpace(value) == "") {
			validationErrors.Add(rule.Field, "Field is required", "")
			continue
		}

		// Skip validation for empty optional fields
		if !exists || value == "" {
			continue
		}

		// Security validation
		if err := iv.ValidateInput(value, context); err != nil {
			validationErrors.Add(rule.Field, err.Error(), value)
			continue
		}

		// Length validation
		if rule.MinLength > 0 && len(value) < rule.MinLength {
			validationErrors.Add(rule.Field, fmt.Sprintf("Minimum length is %d", rule.MinLength), value)
			continue
		}

		if rule.MaxLength > 0 && len(value) > rule.MaxLength {
			validationErrors.Add(rule.Field, fmt.Sprintf("Maximum length is %d", rule.MaxLength), value)
			continue
		}

		// Pattern validation
		if rule.Pattern != "" {
			matched, err := regexp.MatchString(rule.Pattern, value)
			if err != nil {
				validationErrors.Add(rule.Field, "Invalid pattern", value)
				continue
			}
			if !matched {
				validationErrors.Add(rule.Field, "Value does not match required pattern", value)
				continue
			}
		}

		// Allowed values validation
		if len(rule.AllowedValues) > 0 {
			allowed := false
			for _, allowedValue := range rule.AllowedValues {
				if value == allowedValue {
					allowed = true
					break
				}
			}
			if !allowed {
				validationErrors.Add(rule.Field, "Value is not allowed", value)
				continue
			}
		}

		// Custom validation
		if rule.CustomValidator != nil {
			if err := rule.CustomValidator(value); err != nil {
				validationErrors.Add(rule.Field, err.Error(), value)
				continue
			}
		}
	}

	return validationErrors
}

// Security check methods

func (iv *InputValidator) checkSQLInjection(input string) error {
	lowerInput := strings.ToLower(input)

	for _, pattern := range iv.sqlInjectionPatterns {
		if matched, _ := regexp.MatchString(pattern, lowerInput); matched {
			return errors.ErrSecurityViolationError("Potential SQL injection detected")
		}
	}

	return nil
}

func (iv *InputValidator) checkXSS(input string) error {
	lowerInput := strings.ToLower(input)

	for _, pattern := range iv.xssPatterns {
		if matched, _ := regexp.MatchString(pattern, lowerInput); matched {
			return errors.ErrSecurityViolationError("Potential XSS attack detected")
		}
	}

	return nil
}

func (iv *InputValidator) checkCommandInjection(input string) error {
	for _, pattern := range iv.commandPatterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return errors.ErrSecurityViolationError("Potential command injection detected")
		}
	}

	return nil
}

func (iv *InputValidator) checkPathTraversal(input string) error {
	for _, pattern := range iv.pathTraversalPatterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return errors.ErrSecurityViolationError("Potential path traversal detected")
		}
	}

	return nil
}

func (iv *InputValidator) checkSuspiciousPatterns(input string, context *ValidationContext) error {
	// Check for excessive special characters
	specialCharCount := 0
	for _, char := range input {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && !unicode.IsSpace(char) {
			specialCharCount++
		}
	}

	if len(input) > 0 && float64(specialCharCount)/float64(len(input)) > 0.5 {
		return errors.ErrSecurityViolationError("Suspicious character pattern detected")
	}

	// Check for excessively long input
	if len(input) > 10000 {
		return errors.ErrInvalidInputError("Input too long")
	}

	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return errors.ErrSecurityViolationError("Null byte detected")
	}

	return nil
}

// Specific validation functions

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(emailRegex, email)
	if err != nil {
		return errors.NewValidationError("email", "Invalid email format")
	}
	if !matched {
		return errors.NewValidationError("email", "Invalid email format")
	}
	return nil
}

// ValidateURL validates URL format
func ValidateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.NewValidationError("url", "Invalid URL format")
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errors.NewValidationError("url", "URL must include scheme and host")
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.NewValidationError("url", "Only HTTP and HTTPS URLs are allowed")
	}

	return nil
}

// ValidatePhoneNumber validates phone number format
func ValidatePhoneNumber(phone string) error {
	// Remove common separators
	cleanPhone := regexp.MustCompile(`[^0-9+]`).ReplaceAllString(phone, "")

	// Basic phone number validation
	phoneRegex := `^\+?[1-9]\d{1,14}$`
	matched, err := regexp.MatchString(phoneRegex, cleanPhone)
	if err != nil {
		return errors.NewValidationError("phone", "Invalid phone number format")
	}
	if !matched {
		return errors.NewValidationError("phone", "Invalid phone number format")
	}

	return nil
}

// ValidateDate validates date format
func ValidateDate(dateStr, format string) error {
	_, err := time.Parse(format, dateStr)
	if err != nil {
		return errors.NewValidationError("date", "Invalid date format")
	}
	return nil
}

// ValidateInteger validates integer format and range
func ValidateInteger(value string, min, max int) error {
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return errors.NewValidationError("integer", "Invalid integer format")
	}

	if intValue < min || intValue > max {
		return errors.NewValidationError("integer", fmt.Sprintf("Value must be between %d and %d", min, max))
	}

	return nil
}

// ValidateFloat validates float format and range
func ValidateFloat(value string, min, max float64) error {
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errors.NewValidationError("float", "Invalid float format")
	}

	if floatValue < min || floatValue > max {
		return errors.NewValidationError("float", fmt.Sprintf("Value must be between %.2f and %.2f", min, max))
	}

	return nil
}

// ValidateUUID validates UUID format
func ValidateUUID(uuid string) error {
	uuidRegex := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	matched, err := regexp.MatchString(uuidRegex, uuid)
	if err != nil {
		return errors.NewValidationError("uuid", "Invalid UUID format")
	}
	if !matched {
		return errors.NewValidationError("uuid", "Invalid UUID format")
	}
	return nil
}

// SanitizeInput sanitizes input by removing/escaping dangerous characters
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Escape HTML entities
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")

	// Remove control characters
	result := ""
	for _, char := range input {
		if !unicode.IsControl(char) || unicode.IsSpace(char) {
			result += string(char)
		}
	}

	return strings.TrimSpace(result)
}

// Pattern definitions

func getSQLInjectionPatterns() []string {
	return []string{
		`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)\s`,
		`(?i)(or|and)\s+\d+\s*=\s*\d+`,
		`(?i)(or|and)\s+['"]\w+['"]\s*=\s*['"]\w+['"]`,
		`(?i)\b(or|and)\s+1\s*=\s*1\b`,
		`(?i)\b(or|and)\s+['"]?1['"]?\s*=\s*['"]?1['"]?\b`,
		`(?i)\b(union|select)\s+.*\s+from\s+`,
		`(?i)\b(insert|update|delete)\s+.*\s+(into|from|set)\s+`,
		`(?i)\b(drop|create|alter)\s+(table|database|index)\s+`,
		`(?i)\b(exec|execute|sp_|xp_)`,
		`(?i)(--|#|/\*|\*/)`,
		`(?i)\b(information_schema|sysobjects|syscolumns)\b`,
		`(?i)\b(load_file|into\s+outfile|into\s+dumpfile)\b`,
	}
}

func getXSSPatterns() []string {
	return []string{
		`(?i)<script[^>]*>.*?</script>`,
		`(?i)<iframe[^>]*>.*?</iframe>`,
		`(?i)<object[^>]*>.*?</object>`,
		`(?i)<embed[^>]*>`,
		`(?i)<link[^>]*>`,
		`(?i)<meta[^>]*>`,
		`(?i)javascript:`,
		`(?i)vbscript:`,
		`(?i)data:text/html`,
		`(?i)on\w+\s*=`,
		`(?i)<\s*\w+[^>]*\s+on\w+\s*=`,
		`(?i)expression\s*\(`,
		`(?i)@import`,
		`(?i)\\x[0-9a-f]{2}`,
		`(?i)\\u[0-9a-f]{4}`,
	}
}

func getCommandInjectionPatterns() []string {
	return []string{
		`[;&|]\s*(rm|del|format|fdisk)\s`,
		`[;&|]\s*(cat|type|more|less)\s+/`,
		`[;&|]\s*(wget|curl|nc|netcat)\s`,
		`[;&|]\s*(chmod|chown|sudo)\s`,
		`\$\([^)]*\)`,
		"`[^`]*`",
		`\${[^}]*}`,
		`\\x[0-9a-f]{2}`,
	}
}

func getPathTraversalPatterns() []string {
	return []string{
		`\.\./`,
		`\.\.\\`,
		`%2e%2e%2f`,
		`%2e%2e%5c`,
		`%252e%252e%252f`,
		`%c0%ae%c0%ae%c0%af`,
		`\\\.\.\\`,
		`/\.\.\./`,
	}
}
