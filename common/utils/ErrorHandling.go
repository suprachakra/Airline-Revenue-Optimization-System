package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ErrorType represents different categories of errors in the system
type ErrorType string

const (
	// Business Logic Errors
	ValidationError    ErrorType = "VALIDATION_ERROR"
	BusinessRuleError  ErrorType = "BUSINESS_RULE_ERROR"
	DataIntegrityError ErrorType = "DATA_INTEGRITY_ERROR"
	
	// Technical Errors
	DatabaseError      ErrorType = "DATABASE_ERROR"
	NetworkError       ErrorType = "NETWORK_ERROR"
	ServiceUnavailable ErrorType = "SERVICE_UNAVAILABLE"
	TimeoutError       ErrorType = "TIMEOUT_ERROR"
	
	// Security Errors
	AuthenticationError ErrorType = "AUTHENTICATION_ERROR"
	AuthorizationError  ErrorType = "AUTHORIZATION_ERROR"
	SecurityViolation   ErrorType = "SECURITY_VIOLATION"
	
	// External Service Errors
	ExternalAPIError    ErrorType = "EXTERNAL_API_ERROR"
	IntegrationError    ErrorType = "INTEGRATION_ERROR"
	
	// System Errors
	InternalError       ErrorType = "INTERNAL_ERROR"
	ConfigurationError  ErrorType = "CONFIGURATION_ERROR"
	ResourceExhausted   ErrorType = "RESOURCE_EXHAUSTED"
)

// IAROSError represents a standardized error structure across all services
type IAROSError struct {
	// Error Identification
	ID          string    `json:"error_id"`
	Type        ErrorType `json:"error_type"`
	Code        string    `json:"error_code"`
	
	// Error Details
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	UserMessage string    `json:"user_message,omitempty"`
	
	// Context Information
	Service     string    `json:"service"`
	Operation   string    `json:"operation"`
	UserID      string    `json:"user_id,omitempty"`
	RequestID   string    `json:"request_id,omitempty"`
	
	// Technical Details
	Timestamp   time.Time `json:"timestamp"`
	StackTrace  string    `json:"stack_trace,omitempty"`
	Cause       error     `json:"-"`
	
	// HTTP Response Details
	HTTPStatus  int       `json:"http_status"`
	
	// Retry Information
	Retryable   bool      `json:"retryable"`
	RetryAfter  *time.Duration `json:"retry_after,omitempty"`
	
	// Additional Context
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface
func (e *IAROSError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *IAROSError) Unwrap() error {
	return e.Cause
}

// ErrorHandler provides centralized error handling functionality
type ErrorHandler struct {
	logger        *zap.Logger
	metricsCollector ErrorMetricsCollector
	alertingService  AlertingService
	service       string
}

// ErrorMetricsCollector defines interface for error metrics collection
type ErrorMetricsCollector interface {
	RecordError(errorType ErrorType, service, operation string)
	RecordErrorDuration(operation string, duration time.Duration)
	IncrementErrorCount(errorType ErrorType)
}

// AlertingService defines interface for error alerting
type AlertingService interface {
	SendAlert(error *IAROSError, severity string)
	SendSecurityAlert(error *IAROSError)
}

// NewErrorHandler creates a new error handler for a service
func NewErrorHandler(service string, logger *zap.Logger, metrics ErrorMetricsCollector, alerting AlertingService) *ErrorHandler {
	return &ErrorHandler{
		logger:           logger,
		metricsCollector: metrics,
		alertingService:  alerting,
		service:          service,
	}
}

// NewValidationError creates a validation error
func (eh *ErrorHandler) NewValidationError(operation, message string, details ...interface{}) *IAROSError {
	return eh.createError(ValidationError, "VALIDATION_FAILED", operation, message, http.StatusBadRequest, false, details...)
}

// NewBusinessRuleError creates a business rule violation error
func (eh *ErrorHandler) NewBusinessRuleError(operation, message string, details ...interface{}) *IAROSError {
	return eh.createError(BusinessRuleError, "BUSINESS_RULE_VIOLATION", operation, message, http.StatusUnprocessableEntity, false, details...)
}

// NewDatabaseError creates a database error
func (eh *ErrorHandler) NewDatabaseError(operation, message string, cause error) *IAROSError {
	err := eh.createError(DatabaseError, "DATABASE_ERROR", operation, message, http.StatusInternalServerError, true, cause)
	err.Cause = cause
	return err
}

// NewNetworkError creates a network error
func (eh *ErrorHandler) NewNetworkError(operation, message string, cause error) *IAROSError {
	err := eh.createError(NetworkError, "NETWORK_ERROR", operation, message, http.StatusServiceUnavailable, true, cause)
	err.Cause = cause
	err.RetryAfter = durationPtr(5 * time.Second)
	return err
}

// NewAuthenticationError creates an authentication error
func (eh *ErrorHandler) NewAuthenticationError(operation, message string) *IAROSError {
	err := eh.createError(AuthenticationError, "AUTHENTICATION_FAILED", operation, message, http.StatusUnauthorized, false)
	
	// Send security alert for authentication failures
	go eh.alertingService.SendSecurityAlert(err)
	
	return err
}

// NewAuthorizationError creates an authorization error
func (eh *ErrorHandler) NewAuthorizationError(operation, message string) *IAROSError {
	err := eh.createError(AuthorizationError, "AUTHORIZATION_FAILED", operation, message, http.StatusForbidden, false)
	
	// Send security alert for authorization failures
	go eh.alertingService.SendSecurityAlert(err)
	
	return err
}

// NewExternalAPIError creates an external API error
func (eh *ErrorHandler) NewExternalAPIError(operation, message string, cause error) *IAROSError {
	err := eh.createError(ExternalAPIError, "EXTERNAL_API_ERROR", operation, message, http.StatusBadGateway, true, cause)
	err.Cause = cause
	err.RetryAfter = durationPtr(10 * time.Second)
	return err
}

// NewTimeoutError creates a timeout error
func (eh *ErrorHandler) NewTimeoutError(operation, message string) *IAROSError {
	err := eh.createError(TimeoutError, "OPERATION_TIMEOUT", operation, message, http.StatusRequestTimeout, true)
	err.RetryAfter = durationPtr(30 * time.Second)
	return err
}

// NewInternalError creates an internal system error
func (eh *ErrorHandler) NewInternalError(operation, message string, cause error) *IAROSError {
	err := eh.createError(InternalError, "INTERNAL_ERROR", operation, message, http.StatusInternalServerError, false, cause)
	err.Cause = cause
	err.StackTrace = eh.captureStackTrace()
	
	// Send high-priority alert for internal errors
	go eh.alertingService.SendAlert(err, "high")
	
	return err
}

// createError is the central error creation method
func (eh *ErrorHandler) createError(errorType ErrorType, code, operation, message string, httpStatus int, retryable bool, details ...interface{}) *IAROSError {
	errorID := uuid.New().String()
	
	// Create user-friendly message
	userMessage := eh.createUserMessage(errorType, message)
	
	// Format details if provided
	var detailsStr string
	if len(details) > 0 {
		detailsStr = fmt.Sprintf("%v", details)
	}
	
	error := &IAROSError{
		ID:          errorID,
		Type:        errorType,
		Code:        code,
		Message:     message,
		Details:     detailsStr,
		UserMessage: userMessage,
		Service:     eh.service,
		Operation:   operation,
		Timestamp:   time.Now(),
		HTTPStatus:  httpStatus,
		Retryable:   retryable,
		Metadata:    make(map[string]interface{}),
	}
	
	// Log the error
	eh.logError(error)
	
	// Record metrics
	if eh.metricsCollector != nil {
		eh.metricsCollector.RecordError(errorType, eh.service, operation)
		eh.metricsCollector.IncrementErrorCount(errorType)
	}
	
	return error
}

// logError logs the error with appropriate level and context
func (eh *ErrorHandler) logError(err *IAROSError) {
	fields := []zap.Field{
		zap.String("error_id", err.ID),
		zap.String("error_type", string(err.Type)),
		zap.String("error_code", err.Code),
		zap.String("service", err.Service),
		zap.String("operation", err.Operation),
		zap.Int("http_status", err.HTTPStatus),
		zap.Bool("retryable", err.Retryable),
	}
	
	if err.UserID != "" {
		fields = append(fields, zap.String("user_id", err.UserID))
	}
	
	if err.RequestID != "" {
		fields = append(fields, zap.String("request_id", err.RequestID))
	}
	
	if err.Cause != nil {
		fields = append(fields, zap.Error(err.Cause))
	}
	
	// Log based on error severity
	switch err.Type {
	case ValidationError, BusinessRuleError:
		eh.logger.Warn(err.Message, fields...)
	case AuthenticationError, AuthorizationError, SecurityViolation:
		eh.logger.Error(err.Message, fields...)
	case InternalError, DatabaseError:
		eh.logger.Error(err.Message, fields...)
	default:
		eh.logger.Info(err.Message, fields...)
	}
}

// createUserMessage creates user-friendly error messages
func (eh *ErrorHandler) createUserMessage(errorType ErrorType, message string) string {
	switch errorType {
	case ValidationError:
		return "Please check your input and try again."
	case BusinessRuleError:
		return "This operation violates business rules. Please contact support if you need assistance."
	case AuthenticationError:
		return "Authentication failed. Please log in again."
	case AuthorizationError:
		return "You don't have permission to perform this action."
	case NetworkError, ExternalAPIError:
		return "We're experiencing connectivity issues. Please try again in a few moments."
	case TimeoutError:
		return "The operation is taking longer than expected. Please try again."
	case DatabaseError, InternalError:
		return "We're experiencing technical difficulties. Our team has been notified."
	default:
		return "An error occurred. Please try again or contact support."
	}
}

// captureStackTrace captures the current stack trace
func (eh *ErrorHandler) captureStackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// HandleHTTPError handles HTTP error responses consistently
func (eh *ErrorHandler) HandleHTTPError(w http.ResponseWriter, r *http.Request, err *IAROSError) {
	// Set error context from request
	if err.RequestID == "" && r.Header.Get("X-Request-ID") != "" {
		err.RequestID = r.Header.Get("X-Request-ID")
	}
	
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	
	// Don't expose internal details in production
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"id":           err.ID,
			"type":         err.Type,
			"code":         err.Code,
			"message":      err.UserMessage,
			"timestamp":    err.Timestamp,
			"retryable":    err.Retryable,
		},
	}
	
	// Add retry information if applicable
	if err.RetryAfter != nil {
		response["retry_after"] = err.RetryAfter.Seconds()
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", err.RetryAfter.Seconds()))
	}
	
	// Write response
	w.WriteHeader(err.HTTPStatus)
	json.NewEncoder(w).Encode(response)
}

// WrapError wraps an existing error with additional context
func (eh *ErrorHandler) WrapError(err error, operation, message string) *IAROSError {
	if iarosErr, ok := err.(*IAROSError); ok {
		// If it's already an IAROSError, add context
		iarosErr.Operation = operation
		return iarosErr
	}
	
	// Wrap regular error
	return eh.NewInternalError(operation, message, err)
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if iarosErr, ok := err.(*IAROSError); ok {
		return iarosErr.Retryable
	}
	return false
}

// GetRetryAfter returns the retry delay for an error
func GetRetryAfter(err error) *time.Duration {
	if iarosErr, ok := err.(*IAROSError); ok {
		return iarosErr.RetryAfter
	}
	return nil
}

// Helper function to create duration pointer
func durationPtr(d time.Duration) *time.Duration {
	return &d
}

// RecoverFromPanic recovers from panics and converts them to IAROS errors
func (eh *ErrorHandler) RecoverFromPanic(operation string) *IAROSError {
	if r := recover(); r != nil {
		message := fmt.Sprintf("Panic recovered in %s: %v", operation, r)
		err := eh.NewInternalError(operation, message, nil)
		err.StackTrace = eh.captureStackTrace()
		
		// Send critical alert for panics
		go eh.alertingService.SendAlert(err, "critical")
		
		return err
	}
	return nil
}

// Middleware for HTTP request error handling
func (eh *ErrorHandler) ErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add request ID if not present
		if r.Header.Get("X-Request-ID") == "" {
			r.Header.Set("X-Request-ID", uuid.New().String())
		}
		
		// Recover from panics
		defer func() {
			if err := eh.RecoverFromPanic(r.URL.Path); err != nil {
				err.RequestID = r.Header.Get("X-Request-ID")
				eh.HandleHTTPError(w, r, err)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
} 