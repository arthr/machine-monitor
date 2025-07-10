package comms

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"agente-poc/internal/logging"
)

// SecurityManager handles security operations for communications
type SecurityManager struct {
	logger         logging.Logger
	tokenManager   *TokenManager
	certValidator  *CertificateValidator
	rateLimiter    *RateLimiter
	inputSanitizer *InputSanitizer
	config         SecurityConfig
}

// SecurityConfig configuration for security manager
type SecurityConfig struct {
	TokenValidityPeriod  time.Duration
	MaxTokenRefresh      int
	RateLimitWindow      time.Duration
	MaxRequestsPerWindow int
	TLSMinVersion        uint16
	RequiredCipherSuites []uint16
	PinnedCertificates   []string
	AllowedHosts         []string
	Logger               logging.Logger
}

// TokenManager manages authentication tokens
type TokenManager struct {
	tokens         map[string]*Token
	mutex          sync.RWMutex
	logger         logging.Logger
	validityPeriod time.Duration
	maxRefresh     int
}

// Token represents an authentication token
type Token struct {
	Value        string
	IssuedAt     time.Time
	ExpiresAt    time.Time
	RefreshCount int
	Scope        []string
	MachineID    string
}

// CertificateValidator validates TLS certificates
type CertificateValidator struct {
	pinnedCerts  []string
	allowedHosts []string
	logger       logging.Logger
}

// RateLimiter implements rate limiting for requests
type RateLimiter struct {
	requests    map[string]*RequestTracker
	mutex       sync.RWMutex
	windowSize  time.Duration
	maxRequests int
	logger      logging.Logger
}

// RequestTracker tracks requests for rate limiting
type RequestTracker struct {
	Requests  []time.Time
	Blocked   bool
	BlockedAt time.Time
}

// InputSanitizer sanitizes input data
type InputSanitizer struct {
	logger logging.Logger
}

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
	TotalAuthAttempts       int64
	SuccessfulAuth          int64
	FailedAuth              int64
	TokenRefreshes          int64
	RateLimitViolations     int64
	TLSValidationErrors     int64
	InputSanitizationErrors int64
	SecurityAlerts          int64
	LastSecurityEvent       time.Time
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config SecurityConfig) *SecurityManager {
	// Set defaults
	if config.TokenValidityPeriod == 0 {
		config.TokenValidityPeriod = 24 * time.Hour
	}
	if config.MaxTokenRefresh == 0 {
		config.MaxTokenRefresh = 10
	}
	if config.RateLimitWindow == 0 {
		config.RateLimitWindow = 1 * time.Minute
	}
	if config.MaxRequestsPerWindow == 0 {
		config.MaxRequestsPerWindow = 100
	}
	if config.TLSMinVersion == 0 {
		config.TLSMinVersion = tls.VersionTLS12
	}

	tokenManager := &TokenManager{
		tokens:         make(map[string]*Token),
		logger:         config.Logger,
		validityPeriod: config.TokenValidityPeriod,
		maxRefresh:     config.MaxTokenRefresh,
	}

	certValidator := &CertificateValidator{
		pinnedCerts:  config.PinnedCertificates,
		allowedHosts: config.AllowedHosts,
		logger:       config.Logger,
	}

	rateLimiter := &RateLimiter{
		requests:    make(map[string]*RequestTracker),
		windowSize:  config.RateLimitWindow,
		maxRequests: config.MaxRequestsPerWindow,
		logger:      config.Logger,
	}

	inputSanitizer := &InputSanitizer{
		logger: config.Logger,
	}

	return &SecurityManager{
		logger:         config.Logger,
		tokenManager:   tokenManager,
		certValidator:  certValidator,
		rateLimiter:    rateLimiter,
		inputSanitizer: inputSanitizer,
		config:         config,
	}
}

// GenerateToken generates a new authentication token
func (sm *SecurityManager) GenerateToken(machineID string, scope []string) (*Token, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	tokenValue := base64.URLEncoding.EncodeToString(tokenBytes)
	now := time.Now()

	token := &Token{
		Value:        tokenValue,
		IssuedAt:     now,
		ExpiresAt:    now.Add(sm.tokenManager.validityPeriod),
		RefreshCount: 0,
		Scope:        scope,
		MachineID:    machineID,
	}

	sm.tokenManager.mutex.Lock()
	sm.tokenManager.tokens[tokenValue] = token
	sm.tokenManager.mutex.Unlock()

	sm.logger.Info("Generated new token for machine: %s", machineID)
	return token, nil
}

// ValidateToken validates an authentication token
func (sm *SecurityManager) ValidateToken(tokenValue string) (*Token, error) {
	sm.tokenManager.mutex.RLock()
	token, exists := sm.tokenManager.tokens[tokenValue]
	sm.tokenManager.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if time.Now().After(token.ExpiresAt) {
		sm.tokenManager.mutex.Lock()
		delete(sm.tokenManager.tokens, tokenValue)
		sm.tokenManager.mutex.Unlock()
		return nil, fmt.Errorf("token expired")
	}

	return token, nil
}

// RefreshToken refreshes an authentication token
func (sm *SecurityManager) RefreshToken(tokenValue string) (*Token, error) {
	token, err := sm.ValidateToken(tokenValue)
	if err != nil {
		return nil, err
	}

	if token.RefreshCount >= sm.tokenManager.maxRefresh {
		return nil, fmt.Errorf("token refresh limit exceeded")
	}

	sm.tokenManager.mutex.Lock()
	defer sm.tokenManager.mutex.Unlock()

	token.RefreshCount++
	token.ExpiresAt = time.Now().Add(sm.tokenManager.validityPeriod)

	sm.logger.Debug("Token refreshed for machine: %s (refresh count: %d)", token.MachineID, token.RefreshCount)
	return token, nil
}

// RevokeToken revokes an authentication token
func (sm *SecurityManager) RevokeToken(tokenValue string) error {
	sm.tokenManager.mutex.Lock()
	defer sm.tokenManager.mutex.Unlock()

	token, exists := sm.tokenManager.tokens[tokenValue]
	if !exists {
		return fmt.Errorf("token not found")
	}

	delete(sm.tokenManager.tokens, tokenValue)
	sm.logger.Info("Token revoked for machine: %s", token.MachineID)
	return nil
}

// CreateTLSConfig creates a secure TLS configuration
func (sm *SecurityManager) CreateTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:            sm.config.TLSMinVersion,
		CipherSuites:          sm.config.RequiredCipherSuites,
		InsecureSkipVerify:    false,
		VerifyPeerCertificate: sm.certValidator.VerifyPeerCertificate,
		GetCertificate:        sm.certValidator.GetCertificate,
	}
}

// VerifyPeerCertificate verifies peer certificates
func (cv *CertificateValidator) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no certificates provided")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		cv.logger.Error("Failed to parse certificate: %v", err)
		return fmt.Errorf("invalid certificate: %w", err)
	}

	// Check if host is in allowed list
	if len(cv.allowedHosts) > 0 {
		allowed := false
		for _, host := range cv.allowedHosts {
			if cert.Subject.CommonName == host {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("certificate host not allowed: %s", cert.Subject.CommonName)
		}
	}

	// Check pinned certificates
	if len(cv.pinnedCerts) > 0 {
		certHash := sha256.Sum256(cert.Raw)
		certHashStr := hex.EncodeToString(certHash[:])

		for _, pinnedCert := range cv.pinnedCerts {
			if certHashStr == pinnedCert {
				cv.logger.Debug("Certificate pinning verified successfully")
				return nil
			}
		}

		cv.logger.Error("Certificate pinning failed: %s", certHashStr)
		return fmt.Errorf("certificate pinning failed")
	}

	// Standard certificate verification
	roots, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("failed to get system cert pool: %w", err)
	}

	opts := x509.VerifyOptions{
		Roots:       roots,
		CurrentTime: time.Now(),
		DNSName:     cert.Subject.CommonName,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	_, err = cert.Verify(opts)
	if err != nil {
		cv.logger.Error("Certificate verification failed: %v", err)
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	cv.logger.Debug("Certificate verification successful")
	return nil
}

// GetCertificate returns the certificate for the given ClientHelloInfo
func (cv *CertificateValidator) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// This would typically load the appropriate certificate
	// For now, return nil to use default certificate
	return nil, nil
}

// CheckRateLimit checks if a request should be rate limited
func (sm *SecurityManager) CheckRateLimit(identifier string) error {
	sm.rateLimiter.mutex.Lock()
	defer sm.rateLimiter.mutex.Unlock()

	tracker, exists := sm.rateLimiter.requests[identifier]
	if !exists {
		tracker = &RequestTracker{
			Requests: make([]time.Time, 0),
		}
		sm.rateLimiter.requests[identifier] = tracker
	}

	now := time.Now()
	windowStart := now.Add(-sm.rateLimiter.windowSize)

	// Remove old requests
	validRequests := make([]time.Time, 0)
	for _, reqTime := range tracker.Requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	tracker.Requests = validRequests

	// Check if blocked
	if tracker.Blocked && now.Sub(tracker.BlockedAt) < sm.rateLimiter.windowSize {
		return fmt.Errorf("rate limit exceeded, blocked until: %v", tracker.BlockedAt.Add(sm.rateLimiter.windowSize))
	}

	// Check current request count
	if len(tracker.Requests) >= sm.rateLimiter.maxRequests {
		tracker.Blocked = true
		tracker.BlockedAt = now
		sm.logger.Warning("Rate limit exceeded for identifier: %s", identifier)
		return fmt.Errorf("rate limit exceeded")
	}

	// Add current request
	tracker.Requests = append(tracker.Requests, now)
	tracker.Blocked = false

	return nil
}

// SanitizeInput sanitizes input data to prevent injection attacks
func (sm *SecurityManager) SanitizeInput(input string) string {
	return sm.inputSanitizer.Sanitize(input)
}

// Sanitize sanitizes input string
func (is *InputSanitizer) Sanitize(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except newline, tab, and carriage return
	controlChars := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	input = controlChars.ReplaceAllString(input, "")

	// Limit length
	if len(input) > 10000 {
		input = input[:10000]
		is.logger.Warning("Input truncated due to length limit")
	}

	return input
}

// ValidateURL validates and sanitizes URLs
func (sm *SecurityManager) ValidateURL(inputURL string) (*url.URL, error) {
	if inputURL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "wss" {
		return nil, fmt.Errorf("only HTTPS and WSS schemes are allowed")
	}

	// Check host
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("URL must have a host")
	}

	// Check for localhost/private IPs if not allowed
	if strings.Contains(parsedURL.Host, "localhost") || strings.Contains(parsedURL.Host, "127.0.0.1") {
		return nil, fmt.Errorf("localhost URLs are not allowed")
	}

	return parsedURL, nil
}

// AddSecurityHeaders adds security headers to HTTP requests
func (sm *SecurityManager) AddSecurityHeaders(req *http.Request) {
	req.Header.Set("X-Content-Type-Options", "nosniff")
	req.Header.Set("X-Frame-Options", "DENY")
	req.Header.Set("X-XSS-Protection", "1; mode=block")
	req.Header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	req.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
}

// ValidateRequestHeaders validates incoming request headers
func (sm *SecurityManager) ValidateRequestHeaders(headers map[string]string) error {
	for key, value := range headers {
		// Sanitize header key and value
		key = sm.SanitizeInput(key)
		value = sm.SanitizeInput(value)

		// Check for suspicious patterns
		if strings.Contains(strings.ToLower(key), "script") ||
			strings.Contains(strings.ToLower(value), "script") {
			return fmt.Errorf("suspicious header content detected")
		}

		// Check header length
		if len(key) > 100 || len(value) > 1000 {
			return fmt.Errorf("header too long")
		}
	}

	return nil
}

// GenerateNonce generates a cryptographically secure nonce
func (sm *SecurityManager) GenerateNonce() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return base64.URLEncoding.EncodeToString(nonce), nil
}

// HashData creates a SHA-256 hash of data
func (sm *SecurityManager) HashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ValidateJSONPayload validates JSON payload for security issues
func (sm *SecurityManager) ValidateJSONPayload(payload []byte) error {
	// Check payload size
	if len(payload) > 10*1024*1024 { // 10MB limit
		return fmt.Errorf("payload too large")
	}

	// Check for suspicious patterns
	payloadStr := string(payload)
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"eval(",
		"function(",
		"setTimeout(",
		"setInterval(",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(payloadStr), pattern) {
			sm.logger.Warning("Suspicious pattern detected in JSON payload: %s", pattern)
			return fmt.Errorf("suspicious content detected")
		}
	}

	return nil
}

// CleanupExpiredTokens removes expired tokens
func (sm *SecurityManager) CleanupExpiredTokens() {
	sm.tokenManager.mutex.Lock()
	defer sm.tokenManager.mutex.Unlock()

	now := time.Now()
	expiredTokens := make([]string, 0)

	for tokenValue, token := range sm.tokenManager.tokens {
		if now.After(token.ExpiresAt) {
			expiredTokens = append(expiredTokens, tokenValue)
		}
	}

	for _, tokenValue := range expiredTokens {
		delete(sm.tokenManager.tokens, tokenValue)
	}

	if len(expiredTokens) > 0 {
		sm.logger.Info("Cleaned up %d expired tokens", len(expiredTokens))
	}
}

// GetSecurityMetrics returns security metrics
func (sm *SecurityManager) GetSecurityMetrics() SecurityMetrics {
	// This would typically collect real metrics
	// For now, return empty metrics
	return SecurityMetrics{}
}

// IsSecure checks if current configuration is secure
func (sm *SecurityManager) IsSecure() bool {
	// Check TLS version
	if sm.config.TLSMinVersion < tls.VersionTLS12 {
		return false
	}

	// Check if certificate pinning is enabled
	if len(sm.config.PinnedCertificates) == 0 {
		sm.logger.Warning("Certificate pinning is not enabled")
	}

	// Check rate limiting
	if sm.config.MaxRequestsPerWindow <= 0 {
		return false
	}

	return true
}

// StartCleanupRoutine starts the cleanup routine for expired tokens
func (sm *SecurityManager) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.CleanupExpiredTokens()
			}
		}
	}()
}
