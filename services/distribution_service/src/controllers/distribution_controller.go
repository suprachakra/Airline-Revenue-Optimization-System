package controllers

import (
	"net/http"
	"strconv"
	"time"

	"iaros/distribution_service/src/models"
	"iaros/distribution_service/src/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DistributionController handles distribution requests
type DistributionController struct {
	ndcService         *services.NDCService
	gdsService         *services.GDSService
	sessionManager     *services.SessionManager
	transformerService *services.TransformerService
}

// NewDistributionController creates a new distribution controller
func NewDistributionController(
	ndcService *services.NDCService,
	gdsService *services.GDSService,
	sessionManager *services.SessionManager,
	transformerService *services.TransformerService,
) *DistributionController {
	return &DistributionController{
		ndcService:         ndcService,
		gdsService:         gdsService,
		sessionManager:     sessionManager,
		transformerService: transformerService,
	}
}

// ============= NDC Endpoints =============

// ProcessNDCAirShopping handles NDC AirShopping requests
func (dc *DistributionController) ProcessNDCAirShopping(c *gin.Context) {
	var request models.AirShoppingRQ
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Set request ID if not provided
	if request.MessageID == "" {
		request.MessageID = uuid.New().String()
	}

	response, err := dc.ndcService.ProcessAirShopping(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process air shopping request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ProcessNDCOfferPrice handles NDC OfferPrice requests
func (dc *DistributionController) ProcessNDCOfferPrice(c *gin.Context) {
	var request interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := dc.ndcService.ProcessOfferPrice(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process offer price request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ProcessNDCOrderCreate handles NDC OrderCreate requests
func (dc *DistributionController) ProcessNDCOrderCreate(c *gin.Context) {
	var request interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := dc.ndcService.ProcessOrderCreate(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process order create request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ProcessNDCOrderRetrieve handles NDC OrderRetrieve requests
func (dc *DistributionController) ProcessNDCOrderRetrieve(c *gin.Context) {
	var request interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := dc.ndcService.ProcessOrderRetrieve(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process order retrieve request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ProcessNDCOrderCancel handles NDC OrderCancel requests
func (dc *DistributionController) ProcessNDCOrderCancel(c *gin.Context) {
	var request interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := dc.ndcService.ProcessOrderCancel(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process order cancel request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============= GDS Endpoints =============

// ProcessGDSRequest handles GDS requests
func (dc *DistributionController) ProcessGDSRequest(c *gin.Context) {
	var request models.GDSRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Set request ID if not provided
	if request.RequestData["request_id"] == nil {
		request.RequestData["request_id"] = uuid.New().String()
	}

	response, err := dc.gdsService.ProcessGDSRequest(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process GDS request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============= Multi-Channel Distribution =============

// ProcessMultiChannelDistribution handles multi-channel distribution
func (dc *DistributionController) ProcessMultiChannelDistribution(c *gin.Context) {
	var request models.DistributionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Set request ID if not provided
	if request.RequestID == "" {
		request.RequestID = uuid.New().String()
	}

	response, err := dc.transformerService.ProcessMultiChannelDistribution(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process multi-channel distribution",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============= Session Management =============

// CreateNDCSession creates a new NDC session
func (dc *DistributionController) CreateNDCSession(c *gin.Context) {
	customerID := c.Query("customer_id")
	airlineCode := c.Query("airline_code")

	if customerID == "" || airlineCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "customer_id and airline_code are required",
		})
		return
	}

	session, err := dc.sessionManager.CreateNDCSession(c.Request.Context(), customerID, airlineCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create NDC session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetNDCSession retrieves an NDC session
func (dc *DistributionController) GetNDCSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := dc.sessionManager.GetNDCSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Session not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, session)
}

// ExpireSession expires a session
func (dc *DistributionController) ExpireSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	sessionType := c.Query("type")

	if sessionType == "" {
		sessionType = "NDC"
	}

	err := dc.sessionManager.ExpireSession(c.Request.Context(), sessionID, sessionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to expire session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session expired successfully",
	})
}

// GetSessionStats returns session statistics
func (dc *DistributionController) GetSessionStats(c *gin.Context) {
	stats, err := dc.sessionManager.GetSessionStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get session statistics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ============= Health and Monitoring =============

// HealthCheck performs health check
func (dc *DistributionController) HealthCheck(c *gin.Context) {
	gdsHealth := dc.gdsService.HealthCheck(c.Request.Context())
	
	sessionStats, err := dc.sessionManager.GetSessionStats(c.Request.Context())
	if err != nil {
		sessionStats = map[string]interface{}{"error": err.Error()}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "healthy",
		"timestamp":     time.Now().UTC(),
		"gds_health":    gdsHealth,
		"session_stats": sessionStats,
	})
}

// GetMetrics returns service metrics
func (dc *DistributionController) GetMetrics(c *gin.Context) {
	gdsMetrics, err := dc.gdsService.GetGDSMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get GDS metrics",
			"details": err.Error(),
		})
		return
	}

	sessionStats, err := dc.sessionManager.GetSessionStats(c.Request.Context())
	if err != nil {
		sessionStats = map[string]interface{}{"error": err.Error()}
	}

	c.JSON(http.StatusOK, gin.H{
		"gds_metrics":    gdsMetrics,
		"session_stats":  sessionStats,
		"timestamp":      time.Now().UTC(),
	})
}

// ============= Configuration Management =============

// ListChannelConfigurations lists all channel configurations
func (dc *DistributionController) ListChannelConfigurations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	channelType := c.Query("channel_type")
	enabled := c.Query("enabled")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var configurations []models.ChannelConfiguration
	query := dc.transformerService.GetDB().Model(&models.ChannelConfiguration{})

	// Apply filters
	if channelType != "" {
		query = query.Where("channel_type = ?", channelType)
	}
	if enabled != "" {
		enabledBool, _ := strconv.ParseBool(enabled)
		query = query.Where("enabled = ?", enabledBool)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&configurations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list channel configurations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configurations": configurations,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetChannelConfiguration retrieves a specific channel configuration
func (dc *DistributionController) GetChannelConfiguration(c *gin.Context) {
	channelID := c.Param("channel_id")

	var configuration models.ChannelConfiguration
	if err := dc.transformerService.GetDB().Where("channel_id = ?", channelID).First(&configuration).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Channel configuration not found",
		})
		return
	}

	c.JSON(http.StatusOK, configuration)
}

// UpdateChannelConfiguration updates a channel configuration
func (dc *DistributionController) UpdateChannelConfiguration(c *gin.Context) {
	channelID := c.Param("channel_id")

	var configuration models.ChannelConfiguration
	if err := dc.transformerService.GetDB().Where("channel_id = ?", channelID).First(&configuration).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Channel configuration not found",
		})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Update allowed fields
	allowedFields := []string{"name", "description", "enabled", "priority", "configuration", "base_url", "timeout", "retry_attempts"}
	for _, field := range allowedFields {
		if value, exists := updateData[field]; exists {
			switch field {
			case "name":
				configuration.Name = value.(string)
			case "description":
				configuration.Description = value.(string)
			case "enabled":
				configuration.Enabled = value.(bool)
			case "priority":
				configuration.Priority = int(value.(float64))
			case "base_url":
				configuration.BaseURL = value.(string)
			case "timeout":
				configuration.Timeout = time.Duration(value.(float64)) * time.Second
			case "retry_attempts":
				configuration.RetryAttempts = int(value.(float64))
			case "configuration":
				if configData, ok := value.(map[string]interface{}); ok {
					configuration.SetConfiguration(configData)
				}
			}
		}
	}

	if err := dc.transformerService.GetDB().Save(&configuration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update channel configuration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, configuration)
}

// ============= Utility Functions =============

// validateRequest validates common request parameters
func (dc *DistributionController) validateRequest(c *gin.Context) error {
	// Add common validation logic here
	return nil
}

// logRequest logs the incoming request
func (dc *DistributionController) logRequest(c *gin.Context, requestType string) {
	// Add request logging logic here
}

// addCORSHeaders adds CORS headers to the response
func (dc *DistributionController) addCORSHeaders(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
} 