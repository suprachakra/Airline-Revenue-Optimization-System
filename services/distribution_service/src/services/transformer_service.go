package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"iaros/distribution_service/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransformerService handles data transformation between different channels
type TransformerService struct {
	db             *gorm.DB
	sessionManager *SessionManager
}

// NewTransformerService creates a new transformer service
func NewTransformerService(db *gorm.DB, sessionManager *SessionManager) *TransformerService {
	return &TransformerService{
		db:             db,
		sessionManager: sessionManager,
	}
}

// ProcessMultiChannelDistribution processes multi-channel distribution requests
func (ts *TransformerService) ProcessMultiChannelDistribution(ctx context.Context, request *models.DistributionRequest) (*models.DistributionResponse, error) {
	startTime := time.Now()

	// Create distribution transaction
	transaction := &models.DistributionTransaction{
		RequestID:     request.RequestID,
		SourceChannel: request.Channel,
		RequestType:   request.RequestType,
		Status:        models.TransactionStatusPending,
		Priority:      request.Priority,
		RequestedAt:   time.Now(),
		Timeout:       request.Timeout,
	}

	// Set target channels
	if err := transaction.SetTargetChannels(request.TargetChannels); err != nil {
		return nil, fmt.Errorf("failed to set target channels: %w", err)
	}

	// Set request data
	requestData, _ := json.Marshal(request.SourceData)
	transaction.RequestData = string(requestData)

	// Set metadata
	if err := transaction.SetMetadata(request.Metadata); err != nil {
		return nil, fmt.Errorf("failed to set metadata: %w", err)
	}

	// Save transaction
	if err := ts.db.Create(transaction).Error; err != nil {
		return nil, fmt.Errorf("failed to create distribution transaction: %w", err)
	}

	// Update status to processing
	transaction.Status = models.TransactionStatusProcessing
	now := time.Now()
	transaction.ProcessedAt = &now
	ts.db.Save(transaction)

	// Process each target channel
	responses := make(map[string]interface{})
	errors := []models.DistributionError{}
	warnings := []models.DistributionWarning{}

	for _, targetChannel := range request.TargetChannels {
		channelResponse, err := ts.transformAndDistribute(ctx, request, targetChannel, transaction)
		if err != nil {
			errors = append(errors, models.DistributionError{
				Code:    "TRANSFORMATION_ERROR",
				Message: err.Error(),
				Field:   string(targetChannel),
			})
			continue
		}

		responses[string(targetChannel)] = channelResponse
	}

	// Update transaction status
	if len(errors) == 0 {
		transaction.Status = models.TransactionStatusCompleted
	} else if len(responses) > 0 {
		transaction.Status = models.TransactionStatusCompleted
		// Add warnings for partial success
		for _, err := range errors {
			warnings = append(warnings, models.DistributionWarning{
				Code:    err.Code,
				Message: err.Message,
				Field:   err.Field,
			})
		}
		errors = []models.DistributionError{} // Clear errors since we have partial success
	} else {
		transaction.Status = models.TransactionStatusFailed
	}

	completedAt := time.Now()
	transaction.CompletedAt = &completedAt
	transaction.ProcessingTime = time.Since(startTime)

	// Set response data
	responseData, _ := json.Marshal(responses)
	transaction.ResponseData = string(responseData)

	// Set error data if any
	if len(errors) > 0 {
		errorData, _ := json.Marshal(errors)
		transaction.ErrorData = string(errorData)
	}

	ts.db.Save(transaction)

	// Create response
	response := &models.DistributionResponse{
		RequestID:      request.RequestID,
		Channel:        request.Channel,
		Success:        len(errors) == 0,
		Data:           responses,
		Errors:         errors,
		Warnings:       warnings,
		ProcessingTime: time.Since(startTime),
		Metadata: map[string]interface{}{
			"transaction_id":   transaction.TransactionID,
			"channels_processed": len(responses),
			"total_channels":    len(request.TargetChannels),
		},
	}

	return response, nil
}

// transformAndDistribute transforms data and distributes to target channel
func (ts *TransformerService) transformAndDistribute(ctx context.Context, request *models.DistributionRequest, targetChannel models.DistributionChannel, transaction *models.DistributionTransaction) (interface{}, error) {
	// Get transformation rules
	rules, err := ts.getTransformationRules(request.Channel, targetChannel, request.RequestType)
	if err != nil {
		return nil, fmt.Errorf("failed to get transformation rules: %w", err)
	}

	// Transform data
	transformedData, err := ts.transformData(request.SourceData, rules)
	if err != nil {
		return nil, fmt.Errorf("failed to transform data: %w", err)
	}

	// Log the transformation
	ts.logChannelOperation(ctx, transaction.TransactionID, string(targetChannel), "TRANSFORM", transformedData, nil)

	// Mock distribution (in real implementation, this would call the actual channel APIs)
	response := ts.mockChannelDistribution(targetChannel, transformedData)

	// Log the distribution
	ts.logChannelOperation(ctx, transaction.TransactionID, string(targetChannel), "DISTRIBUTE", transformedData, response)

	return response, nil
}

// getTransformationRules retrieves transformation rules for source and target channels
func (ts *TransformerService) getTransformationRules(sourceChannel, targetChannel models.DistributionChannel, messageType string) ([]models.TransformationRule, error) {
	var rules []models.TransformationRule

	err := ts.db.Where("source_channel = ? AND target_channel = ? AND message_type = ? AND enabled = ?",
		sourceChannel, targetChannel, messageType, true).
		Order("priority ASC").
		Find(&rules).Error

	if err != nil {
		return nil, err
	}

	return rules, nil
}

// transformData applies transformation rules to the source data
func (ts *TransformerService) transformData(sourceData map[string]interface{}, rules []models.TransformationRule) (map[string]interface{}, error) {
	transformedData := make(map[string]interface{})

	// Copy source data as starting point
	for key, value := range sourceData {
		transformedData[key] = value
	}

	// Apply each transformation rule
	for _, rule := range rules {
		if err := ts.applyTransformationRule(transformedData, rule); err != nil {
			return nil, fmt.Errorf("failed to apply rule %s: %w", rule.Name, err)
		}

		// Update rule execution statistics
		ts.updateRuleExecutionStats(rule.RuleID, true)
	}

	return transformedData, nil
}

// applyTransformationRule applies a single transformation rule
func (ts *TransformerService) applyTransformationRule(data map[string]interface{}, rule models.TransformationRule) error {
	// Parse field mappings
	var fieldMappings []models.FieldMapping
	if rule.FieldMappings != "" {
		if err := json.Unmarshal([]byte(rule.FieldMappings), &fieldMappings); err != nil {
			return fmt.Errorf("failed to parse field mappings: %w", err)
		}
	}

	// Apply field mappings
	for _, mapping := range fieldMappings {
		value := ts.getNestedValue(data, mapping.SourceField)
		if value == nil && mapping.DefaultValue != nil {
			value = mapping.DefaultValue
		}

		if value != nil || mapping.Required {
			ts.setNestedValue(data, mapping.TargetField, value)
		}
	}

	// Parse and apply conditions
	var conditions []models.TransformCondition
	if rule.Conditions != "" {
		if err := json.Unmarshal([]byte(rule.Conditions), &conditions); err != nil {
			return fmt.Errorf("failed to parse conditions: %w", err)
		}
	}

	for _, condition := range conditions {
		if ts.evaluateCondition(data, condition) {
			ts.applyConditionAction(data, condition)
		}
	}

	return nil
}

// getNestedValue retrieves a nested value from a map using dot notation
func (ts *TransformerService) getNestedValue(data map[string]interface{}, path string) interface{} {
	// Simple implementation - in production, use a proper JSON path library
	return data[path]
}

// setNestedValue sets a nested value in a map using dot notation
func (ts *TransformerService) setNestedValue(data map[string]interface{}, path string, value interface{}) {
	// Simple implementation - in production, use a proper JSON path library
	data[path] = value
}

// evaluateCondition evaluates a transformation condition
func (ts *TransformerService) evaluateCondition(data map[string]interface{}, condition models.TransformCondition) bool {
	fieldValue := ts.getNestedValue(data, condition.Field)
	
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "exists":
		return fieldValue != nil
	case "not_exists":
		return fieldValue == nil
	default:
		return false
	}
}

// applyConditionAction applies the action specified in a condition
func (ts *TransformerService) applyConditionAction(data map[string]interface{}, condition models.TransformCondition) {
	switch condition.Action {
	case "set":
		ts.setNestedValue(data, condition.Field, condition.Value)
	case "remove":
		delete(data, condition.Field)
	case "copy":
		// Implementation for copy action
	default:
		// Unknown action
	}
}

// mockChannelDistribution simulates distribution to different channels
func (ts *TransformerService) mockChannelDistribution(channel models.DistributionChannel, data map[string]interface{}) map[string]interface{} {
	response := map[string]interface{}{
		"channel":     string(channel),
		"success":     true,
		"timestamp":   time.Now().UTC(),
		"request_id":  uuid.New().String(),
	}

	switch channel {
	case models.NDCChannel:
		response["ndc_response"] = map[string]interface{}{
			"message_id": uuid.New().String(),
			"version":    "20.3",
			"status":     "SUCCESS",
			"offers":     []string{"OFFER_001", "OFFER_002"},
		}
	case models.GDSChannel:
		response["gds_response"] = map[string]interface{}{
			"session_id":    uuid.New().String(),
			"status":        "SUCCESS",
			"itineraries":   []string{"ITIN_001", "ITIN_002"},
			"provider":      "AMADEUS",
		}
	case models.OTAChannel:
		response["ota_response"] = map[string]interface{}{
			"booking_id":    uuid.New().String(),
			"status":        "CONFIRMED",
			"confirmation":  "ABC123",
		}
	case models.DirectChannel:
		response["direct_response"] = map[string]interface{}{
			"order_id":      uuid.New().String(),
			"status":        "CREATED",
			"payment_link":  "https://payment.airline.com/pay/123",
		}
	default:
		response["generic_response"] = map[string]interface{}{
			"status": "PROCESSED",
			"data":   data,
		}
	}

	return response
}

// logChannelOperation logs channel operations for audit trail
func (ts *TransformerService) logChannelOperation(ctx context.Context, transactionID, channelID, operationType string, requestData, responseData interface{}) {
	log := &models.ChannelLog{
		TransactionID: transactionID,
		ChannelID:     channelID,
		ChannelType:   models.DistributionChannel(channelID),
		RequestType:   operationType,
		StartedAt:     time.Now(),
		Success:       responseData != nil,
	}

	if requestData != nil {
		requestJSON, _ := json.Marshal(requestData)
		log.RequestBody = string(requestJSON)
	}

	if responseData != nil {
		responseJSON, _ := json.Marshal(responseData)
		log.ResponseBody = string(responseJSON)
		completedAt := time.Now()
		log.CompletedAt = &completedAt
		log.Duration = time.Since(log.StartedAt)
	}

	ts.db.Create(log)
}

// updateRuleExecutionStats updates transformation rule execution statistics
func (ts *TransformerService) updateRuleExecutionStats(ruleID string, success bool) {
	var rule models.TransformationRule
	if err := ts.db.Where("rule_id = ?", ruleID).First(&rule).Error; err != nil {
		return
	}

	rule.ExecutionCount++
	if success {
		rule.SuccessCount++
	} else {
		rule.ErrorCount++
	}

	now := time.Now()
	rule.LastExecuted = &now

	ts.db.Save(&rule)
}

// GetDB returns the database instance for use in controllers
func (ts *TransformerService) GetDB() *gorm.DB {
	return ts.db
}

// GetChannelConfiguration retrieves channel configuration
func (ts *TransformerService) GetChannelConfiguration(channelID string) (*models.ChannelConfiguration, error) {
	var config models.ChannelConfiguration
	err := ts.db.Where("channel_id = ?", channelID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateChannelConfiguration updates channel configuration
func (ts *TransformerService) UpdateChannelConfiguration(config *models.ChannelConfiguration) error {
	return ts.db.Save(config).Error
}

// GetTransformationRulesByChannel gets transformation rules for a channel
func (ts *TransformerService) GetTransformationRulesByChannel(sourceChannel, targetChannel models.DistributionChannel) ([]models.TransformationRule, error) {
	var rules []models.TransformationRule
	err := ts.db.Where("source_channel = ? AND target_channel = ? AND enabled = ?",
		sourceChannel, targetChannel, true).
		Order("priority ASC").
		Find(&rules).Error
	return rules, err
}

// CreateTransformationRule creates a new transformation rule
func (ts *TransformerService) CreateTransformationRule(rule *models.TransformationRule) error {
	return ts.db.Create(rule).Error
}

// UpdateTransformationRule updates a transformation rule
func (ts *TransformerService) UpdateTransformationRule(rule *models.TransformationRule) error {
	return ts.db.Save(rule).Error
}

// DeleteTransformationRule deletes a transformation rule
func (ts *TransformerService) DeleteTransformationRule(ruleID string) error {
	return ts.db.Where("rule_id = ?", ruleID).Delete(&models.TransformationRule{}).Error
} 