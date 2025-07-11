package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// HTTPClient represents an HTTP client with advanced features
type HTTPClient struct {
	client         *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
	config         Config
	logger         *zap.Logger
}

// Config holds configuration for HTTP client
type Config struct {
	Timeout         time.Duration
	Retries         int
	CircuitBreaker  bool
	RetryInterval   time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
	UserAgent       string
}

// Response represents an HTTP response
type Response struct {
	*http.Response
	Body []byte
}

// NewHTTPClient creates a new HTTP client with the specified configuration
func NewHTTPClient(config Config) *HTTPClient {
	// Set default values
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Retries == 0 {
		config.Retries = 3
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = 1 * time.Second
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.MaxConnsPerHost == 0 {
		config.MaxConnsPerHost = 10
	}
	if config.UserAgent == "" {
		config.UserAgent = "IAROS-Client/1.0"
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		MaxIdleConns:       config.MaxIdleConns,
		MaxConnsPerHost:    config.MaxConnsPerHost,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: false,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	// Initialize logger
	logger, _ := zap.NewProduction()

	client := &HTTPClient{
		client: httpClient,
		config: config,
		logger: logger,
	}

	// Initialize circuit breaker if enabled
	if config.CircuitBreaker {
		cbConfig := gobreaker.Settings{
			Name:        "HTTP-Client",
			MaxRequests: 3,
			Interval:    10 * time.Second,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures > 2
			},
			OnStateChange: func(name string, from, to gobreaker.State) {
				logger.Info("Circuit breaker state changed",
					zap.String("name", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			},
		}
		client.circuitBreaker = gobreaker.NewCircuitBreaker(cbConfig)
	}

	return client
}

// Get performs a GET request
func (c *HTTPClient) Get(url string, headers ...map[string]string) (*Response, error) {
	return c.request(http.MethodGet, url, nil, headers...)
}

// Post performs a POST request
func (c *HTTPClient) Post(url string, body interface{}, headers ...map[string]string) (*Response, error) {
	return c.request(http.MethodPost, url, body, headers...)
}

// Put performs a PUT request
func (c *HTTPClient) Put(url string, body interface{}, headers ...map[string]string) (*Response, error) {
	return c.request(http.MethodPut, url, body, headers...)
}

// Delete performs a DELETE request
func (c *HTTPClient) Delete(url string, headers ...map[string]string) (*Response, error) {
	return c.request(http.MethodDelete, url, nil, headers...)
}

// Patch performs a PATCH request
func (c *HTTPClient) Patch(url string, body interface{}, headers ...map[string]string) (*Response, error) {
	return c.request(http.MethodPatch, url, body, headers...)
}

// request performs the actual HTTP request with retries and circuit breaker
func (c *HTTPClient) request(method, url string, body interface{}, headers ...map[string]string) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.Retries; attempt++ {
		// Create request body
		var requestBody io.Reader
		if body != nil {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			requestBody = bytes.NewReader(bodyBytes)
		}

		// Create request
		req, err := http.NewRequest(method, url, requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", c.config.UserAgent)
		req.Header.Set("Accept", "application/json")

		// Add custom headers
		for _, headerMap := range headers {
			for key, value := range headerMap {
				req.Header.Set(key, value)
			}
		}

		// Execute request with circuit breaker
		var resp *http.Response
		if c.circuitBreaker != nil {
			result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
				return c.client.Do(req)
			})
			if err != nil {
				lastErr = err
				c.logger.Warn("Request failed",
					zap.String("method", method),
					zap.String("url", url),
					zap.Int("attempt", attempt+1),
					zap.Error(err),
				)
				if attempt < c.config.Retries {
					time.Sleep(c.config.RetryInterval * time.Duration(attempt+1))
				}
				continue
			}
			resp = result.(*http.Response)
		} else {
			resp, err = c.client.Do(req)
			if err != nil {
				lastErr = err
				c.logger.Warn("Request failed",
					zap.String("method", method),
					zap.String("url", url),
					zap.Int("attempt", attempt+1),
					zap.Error(err),
				)
				if attempt < c.config.Retries {
					time.Sleep(c.config.RetryInterval * time.Duration(attempt+1))
				}
				continue
			}
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			if attempt < c.config.Retries {
				time.Sleep(c.config.RetryInterval * time.Duration(attempt+1))
			}
			continue
		}
		resp.Body.Close()

		// Check for HTTP errors
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
			if attempt < c.config.Retries && resp.StatusCode >= 500 {
				c.logger.Warn("Server error, retrying",
					zap.String("method", method),
					zap.String("url", url),
					zap.Int("status", resp.StatusCode),
					zap.Int("attempt", attempt+1),
				)
				time.Sleep(c.config.RetryInterval * time.Duration(attempt+1))
				continue
			}
			// Don't retry client errors (4xx)
			if resp.StatusCode < 500 {
				break
			}
		}

		// Success
		c.logger.Debug("Request successful",
			zap.String("method", method),
			zap.String("url", url),
			zap.Int("status", resp.StatusCode),
			zap.Int("attempt", attempt+1),
		)

		return &Response{
			Response: resp,
			Body:     respBody,
		}, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.Retries+1, lastErr)
}

// GetJSON performs a GET request and unmarshals the response into the provided interface
func (c *HTTPClient) GetJSON(url string, target interface{}, headers ...map[string]string) error {
	resp, err := c.Get(url, headers...)
	if err != nil {
		return err
	}

	return json.Unmarshal(resp.Body, target)
}

// PostJSON performs a POST request and unmarshals the response into the provided interface
func (c *HTTPClient) PostJSON(url string, body interface{}, target interface{}, headers ...map[string]string) error {
	resp, err := c.Post(url, body, headers...)
	if err != nil {
		return err
	}

	return json.Unmarshal(resp.Body, target)
}

// PutJSON performs a PUT request and unmarshals the response into the provided interface
func (c *HTTPClient) PutJSON(url string, body interface{}, target interface{}, headers ...map[string]string) error {
	resp, err := c.Put(url, body, headers...)
	if err != nil {
		return err
	}

	return json.Unmarshal(resp.Body, target)
}

// WithContext creates a new client with the specified context
func (c *HTTPClient) WithContext(ctx context.Context) *HTTPClient {
	newClient := *c
	newClient.client = &http.Client{
		Transport: c.client.Transport,
		Timeout:   c.config.Timeout,
	}
	// Note: Context would be applied to individual requests
	return &newClient
}

// SetTimeout sets the timeout for the client
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.config.Timeout = timeout
	c.client.Timeout = timeout
}

// SetRetries sets the number of retries for the client
func (c *HTTPClient) SetRetries(retries int) {
	c.config.Retries = retries
}

// Close closes the HTTP client and cleans up resources
func (c *HTTPClient) Close() error {
	if c.client != nil && c.client.Transport != nil {
		if transport, ok := c.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}

// GetStats returns statistics about the client
func (c *HTTPClient) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"timeout":           c.config.Timeout,
		"retries":           c.config.Retries,
		"circuit_breaker":   c.config.CircuitBreaker,
		"retry_interval":    c.config.RetryInterval,
		"max_idle_conns":    c.config.MaxIdleConns,
		"max_conns_per_host": c.config.MaxConnsPerHost,
	}

	if c.circuitBreaker != nil {
		cbStats := c.circuitBreaker.Counts()
		stats["circuit_breaker_stats"] = map[string]interface{}{
			"state":               c.circuitBreaker.State().String(),
			"requests":            cbStats.Requests,
			"total_successes":     cbStats.TotalSuccesses,
			"total_failures":      cbStats.TotalFailures,
			"consecutive_successes": cbStats.ConsecutiveSuccesses,
			"consecutive_failures":  cbStats.ConsecutiveFailures,
		}
	}

	return stats
} 