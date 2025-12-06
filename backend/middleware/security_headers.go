package middleware

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SecurityHeadersConfig defines configuration for security headers middleware
type SecurityHeadersConfig struct {
	// Skipper defines a function to skip middleware
	Skipper middleware.Skipper

	// ContentTypeNosniff sets X-Content-Type-Options header to "nosniff"
	ContentTypeNosniff bool

	// FrameDeny sets X-Frame-Options header to "DENY"
	FrameDeny bool

	// FrameOptions allows custom X-Frame-Options values ("DENY", "SAMEORIGIN", etc.)
	FrameOptions string

	// XSSProtection sets X-XSS-Protection header
	XSSProtection string

	// ContentSecurityPolicy sets Content-Security-Policy header
	ContentSecurityPolicy string

	// ReferrerPolicy sets Referrer-Policy header
	ReferrerPolicy string

	// HSTS enables HTTP Strict Transport Security
	HSTS HSSTConfig

	// PermissionsPolicy sets Permissions-Policy header (formerly Feature-Policy)
	PermissionsPolicy string

	// CrossOriginEmbedderPolicy sets Cross-Origin-Embedder-Policy header
	CrossOriginEmbedderPolicy string

	// CrossOriginOpenerPolicy sets Cross-Origin-Opener-Policy header
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy sets Cross-Origin-Resource-Policy header
	CrossOriginResourcePolicy string

	// Custom headers to add
	CustomHeaders map[string]string
}

// HSSTConfig defines HSTS configuration
type HSSTConfig struct {
	// Enabled enables HSTS
	Enabled bool

	// MaxAge sets max-age directive (in seconds)
	MaxAge int

	// IncludeSubdomains adds includeSubDomains directive
	IncludeSubdomains bool

	// Preload adds preload directive
	Preload bool
}

// DefaultSecurityHeadersConfig provides secure defaults
var DefaultSecurityHeadersConfig = SecurityHeadersConfig{
	Skipper:            middleware.DefaultSkipper,
	ContentTypeNosniff: true,
	FrameOptions:       "DENY",
	XSSProtection:      "1; mode=block",
	ContentSecurityPolicy: strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data: https:",
		"connect-src 'self' https://api.* wss://api.*",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}, "; "),
	ReferrerPolicy: "strict-origin-when-cross-origin",
	HSTS: HSSTConfig{
		Enabled:           true,
		MaxAge:            31536000, // 1 year
		IncludeSubdomains: true,
		Preload:           false, // Set to true only after careful consideration
	},
	PermissionsPolicy: strings.Join([]string{
		"camera=()",
		"microphone=()",
		"geolocation=()",
		"interest-cohort=()",
	}, ", "),
	CrossOriginEmbedderPolicy:  "require-corp",
	CrossOriginOpenerPolicy:    "same-origin",
	CrossOriginResourcePolicy: "cross-origin",
}

// SecurityHeadersWithConfig returns security headers middleware with configuration
func SecurityHeadersWithConfig(config SecurityHeadersConfig) echo.MiddlewareFunc {
	// Set defaults
	if config.Skipper == nil {
		config.Skipper = DefaultSecurityHeadersConfig.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			res := c.Response()

			// X-Content-Type-Options: nosniff
			if config.ContentTypeNosniff {
				res.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// X-Frame-Options
			if config.FrameOptions != "" {
				res.Header().Set("X-Frame-Options", config.FrameOptions)
			} else if config.FrameDeny {
				res.Header().Set("X-Frame-Options", "DENY")
			}

			// X-XSS-Protection
			if config.XSSProtection != "" {
				res.Header().Set("X-XSS-Protection", config.XSSProtection)
			}

			// Content-Security-Policy
			if config.ContentSecurityPolicy != "" {
				res.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				res.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// HTTP Strict Transport Security
			if config.HSTS.Enabled {
				hstsValue := fmt.Sprintf("max-age=%d", config.HSTS.MaxAge)
				if config.HSTS.IncludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				if config.HSTS.Preload {
					hstsValue += "; preload"
				}
				res.Header().Set("Strict-Transport-Security", hstsValue)
			}

			// Permissions-Policy
			if config.PermissionsPolicy != "" {
				res.Header().Set("Permissions-Policy", config.PermissionsPolicy)
			}

			// Cross-Origin-Embedder-Policy
			if config.CrossOriginEmbedderPolicy != "" {
				res.Header().Set("Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
			}

			// Cross-Origin-Opener-Policy
			if config.CrossOriginOpenerPolicy != "" {
				res.Header().Set("Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			}

			// Cross-Origin-Resource-Policy
			if config.CrossOriginResourcePolicy != "" {
				res.Header().Set("Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			}

			// Custom headers
			for key, value := range config.CustomHeaders {
				res.Header().Set(key, value)
			}

			return next(c)
		}
	}
}

// SecurityHeaders returns security headers middleware with default configuration
func SecurityHeaders() echo.MiddlewareFunc {
	return SecurityHeadersWithConfig(DefaultSecurityHeadersConfig)
}

// ProductionSecurityHeaders returns security headers middleware optimized for production
func ProductionSecurityHeaders(domain string) echo.MiddlewareFunc {
	config := DefaultSecurityHeadersConfig
	
	// More restrictive CSP for production
	config.ContentSecurityPolicy = strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data: https:",
		fmt.Sprintf("connect-src 'self' https://api.%s wss://api.%s", domain, domain),
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"upgrade-insecure-requests",
	}, "; ")
	
	// Enable HSTS preload for production (after careful consideration)
	config.HSTS.Preload = true
	config.HSTS.MaxAge = 63072000 // 2 years
	
	// More restrictive cross-origin policies
	config.CrossOriginEmbedderPolicy = "require-corp"
	config.CrossOriginOpenerPolicy = "same-origin"
	
	return SecurityHeadersWithConfig(config)
}

// DevelopmentSecurityHeaders returns security headers middleware optimized for development
func DevelopmentSecurityHeaders() echo.MiddlewareFunc {
	config := DefaultSecurityHeadersConfig
	
	// More lenient CSP for development
	config.ContentSecurityPolicy = strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data: https: http:",
		"connect-src 'self' https: http: ws: wss:",
		"frame-ancestors 'self'",
		"base-uri 'self'",
		"form-action 'self'",
	}, "; ")
	
	// Disable HSTS in development
	config.HSTS.Enabled = false
	
	// More lenient frame options
	config.FrameOptions = "SAMEORIGIN"
	
	// Less restrictive cross-origin policies
	config.CrossOriginEmbedderPolicy = ""
	config.CrossOriginResourcePolicy = "cross-origin"
	
	return SecurityHeadersWithConfig(config)
}

// APISecurityHeaders returns security headers middleware optimized for API endpoints
func APISecurityHeaders() echo.MiddlewareFunc {
	config := SecurityHeadersConfig{
		Skipper:            middleware.DefaultSkipper,
		ContentTypeNosniff: true,
		FrameOptions:       "DENY",
		XSSProtection:      "1; mode=block",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		HSTS: HSSTConfig{
			Enabled:           true,
			MaxAge:            31536000,
			IncludeSubdomains: true,
			Preload:           false,
		},
		// No CSP for API endpoints as they don't serve HTML
		CrossOriginResourcePolicy: "cross-origin",
		CustomHeaders: map[string]string{
			"X-API-Version": "1.0",
			"Cache-Control": "no-store, no-cache, must-revalidate, private",
		},
	}
	
	return SecurityHeadersWithConfig(config)
}

// HealthCheckSecurityHeaders returns minimal security headers for health check endpoints
func HealthCheckSecurityHeaders() echo.MiddlewareFunc {
	config := SecurityHeadersConfig{
		Skipper:            middleware.DefaultSkipper,
		ContentTypeNosniff: true,
		CustomHeaders: map[string]string{
			"Cache-Control": "no-cache, no-store, must-revalidate",
			"Pragma":        "no-cache",
			"Expires":       "0",
		},
	}
	
	return SecurityHeadersWithConfig(config)
}

// RemoveServerHeader removes or replaces the Server header
func RemoveServerHeader() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Remove server header to avoid information disclosure
			c.Response().Header().Set("Server", "")
			return next(c)
		}
	}
}

// CustomSecurityHeaders allows easy creation of custom security header configurations
func CustomSecurityHeaders(options map[string]interface{}) echo.MiddlewareFunc {
	config := DefaultSecurityHeadersConfig
	
	// Override defaults with provided options
	if csp, ok := options["csp"].(string); ok && csp != "" {
		config.ContentSecurityPolicy = csp
	}
	
	if frameOptions, ok := options["frame_options"].(string); ok && frameOptions != "" {
		config.FrameOptions = frameOptions
	}
	
	if referrerPolicy, ok := options["referrer_policy"].(string); ok && referrerPolicy != "" {
		config.ReferrerPolicy = referrerPolicy
	}
	
	if hstsEnabled, ok := options["hsts_enabled"].(bool); ok {
		config.HSTS.Enabled = hstsEnabled
	}
	
	if hstsMaxAge, ok := options["hsts_max_age"].(int); ok && hstsMaxAge > 0 {
		config.HSTS.MaxAge = hstsMaxAge
	}
	
	if customHeaders, ok := options["custom_headers"].(map[string]string); ok {
		config.CustomHeaders = customHeaders
	}
	
	return SecurityHeadersWithConfig(config)
}