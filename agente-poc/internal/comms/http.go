package comms

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agente-poc/internal/logging"
)

// HTTPClient wraps the HTTP client with retry, authentication and monitoring
type HTTPClient struct {
	client    *http.Client
	baseURL   string
	token     string
	userAgent string
	logger    logging.Logger
	metrics   *HTTPMetrics
}

// HTTPMetrics tracks HTTP client metrics
type HTTPMetrics struct {
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	RetryCount       int64
	AverageLatency   time.Duration
	LastRequestTime  time.Time
	TotalBytes       int64
	ConnectionErrors int64
}

// HTTPConfig configuration for HTTP client
type HTTPConfig struct {
	BaseURL         string
	Token           string
	UserAgent       string
	Timeout         time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	MaxRetryDelay   time.Duration
	TLSSkipVerify   bool
	ConnectTimeout  time.Duration
	IdleTimeout     time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
	Logger          logging.Logger
}

// NewHTTPClient creates a new HTTP client with the given configuration
func NewHTTPClient(config HTTPConfig) *HTTPClient {
	// Create custom transport with timeouts and connection pooling
	transport := &http.Transport{
		MaxIdleConns:       config.MaxIdleConns,
		MaxConnsPerHost:    config.MaxConnsPerHost,
		IdleConnTimeout:    config.IdleTimeout,
		DisableCompression: false,
		ForceAttemptHTTP2:  true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.TLSSkipVerify,
		},
	}

	// Create HTTP client with custom transport
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &HTTPClient{
		client:    client,
		baseURL:   config.BaseURL,
		token:     config.Token,
		userAgent: config.UserAgent,
		logger:    config.Logger,
		metrics:   &HTTPMetrics{},
	}
}

// sendRequest sends an HTTP request with retry logic
func (c *HTTPClient) sendRequest(ctx context.Context, method, endpoint string, body interface{}, target interface{}) error {
	var jsonBody []byte
	var err error

	// Serialize body if provided
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := c.baseURL + endpoint
	maxRetries := 3
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create request
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")

		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		// Add security headers
		req.Header.Set("X-Request-ID", fmt.Sprintf("%d", time.Now().UnixNano()))
		req.Header.Set("X-Agent-Version", "1.0.0")

		// Record metrics
		c.metrics.TotalRequests++
		c.metrics.LastRequestTime = time.Now()
		startTime := time.Now()

		// Send request
		resp, err := c.client.Do(req)
		if err != nil {
			c.metrics.FailedRequests++
			c.metrics.ConnectionErrors++

			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * baseDelay
				c.logger.WithFields(map[string]interface{}{
					"attempt": attempt + 1,
					"delay":   delay,
					"error":   err.Error(),
					"url":     url,
				}).Warning("HTTP request failed, retrying...")

				c.metrics.RetryCount++

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}

			return fmt.Errorf("HTTP request failed after %d attempts: %w", maxRetries+1, err)
		}

		// Update metrics
		latency := time.Since(startTime)
		c.metrics.AverageLatency = (c.metrics.AverageLatency + latency) / 2

		// Read response body
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			c.metrics.FailedRequests++
			return fmt.Errorf("failed to read response body: %w", err)
		}

		c.metrics.TotalBytes += int64(len(bodyBytes))

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			c.metrics.SuccessRequests++

			// Parse response if target is provided
			if target != nil && len(bodyBytes) > 0 {
				if err := json.Unmarshal(bodyBytes, target); err != nil {
					return fmt.Errorf("failed to unmarshal response: %w", err)
				}
			}

			c.logger.WithFields(map[string]interface{}{
				"method":      method,
				"endpoint":    endpoint,
				"status_code": resp.StatusCode,
				"latency":     latency,
				"size":        len(bodyBytes),
			}).Debug("HTTP request successful")

			return nil
		}

		// Handle error responses
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Client errors - don't retry
			c.metrics.FailedRequests++

			var errorResp ErrorResponse
			if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
				return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, errorResp.Message)
			}

			return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(bodyBytes))
		}

		// Server errors - retry if possible
		if resp.StatusCode >= 500 && attempt < maxRetries {
			delay := time.Duration(attempt+1) * baseDelay
			c.logger.WithFields(map[string]interface{}{
				"attempt":     attempt + 1,
				"delay":       delay,
				"status_code": resp.StatusCode,
				"url":         url,
			}).Warning("HTTP server error, retrying...")

			c.metrics.RetryCount++

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		c.metrics.FailedRequests++
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return fmt.Errorf("HTTP request failed after %d attempts", maxRetries+1)
}

// GET performs a GET request
func (c *HTTPClient) GET(ctx context.Context, endpoint string, target interface{}) error {
	return c.sendRequest(ctx, "GET", endpoint, nil, target)
}

// POST performs a POST request
func (c *HTTPClient) POST(ctx context.Context, endpoint string, body interface{}, target interface{}) error {
	return c.sendRequest(ctx, "POST", endpoint, body, target)
}

// PUT performs a PUT request
func (c *HTTPClient) PUT(ctx context.Context, endpoint string, body interface{}, target interface{}) error {
	return c.sendRequest(ctx, "PUT", endpoint, body, target)
}

// DELETE performs a DELETE request
func (c *HTTPClient) DELETE(ctx context.Context, endpoint string, target interface{}) error {
	return c.sendRequest(ctx, "DELETE", endpoint, nil, target)
}

// GetMetrics returns the current HTTP client metrics
func (c *HTTPClient) GetMetrics() HTTPMetrics {
	return *c.metrics
}

// ResetMetrics resets the HTTP client metrics
func (c *HTTPClient) ResetMetrics() {
	c.metrics = &HTTPMetrics{}
}

// IsHealthy checks if the HTTP client is healthy
func (c *HTTPClient) IsHealthy() bool {
	if c.metrics.TotalRequests == 0 {
		return true // No requests yet
	}

	successRate := float64(c.metrics.SuccessRequests) / float64(c.metrics.TotalRequests)
	return successRate >= 0.8 // 80% success rate threshold
}

// Close closes the HTTP client and cleans up resources
func (c *HTTPClient) Close() error {
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}
