package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PartnerDashboardService struct {
	db          *mongo.Database
	wsClients   map[string]*websocket.Conn
	wsUpgrader  websocket.Upgrader
	analytics   *AnalyticsEngine
	realTime    *RealTimeMetrics
}

type DashboardSummary struct {
	TotalRevenue     float64 `json:"totalRevenue"`
	TotalBookings    int64   `json:"totalBookings"`
	ConversionRate   float64 `json:"conversionRate"`
	HealthScore      float64 `json:"healthScore"`
	RevenueChange    float64 `json:"revenueChange"`
	BookingsChange   float64 `json:"bookingsChange"`
	ConversionChange float64 `json:"conversionChange"`
}

type RevenueData struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
}

type BookingTrend struct {
	Date     string `json:"date"`
	Bookings int64  `json:"bookings"`
	Searches int64  `json:"searches"`
}

type RoutePerformance struct {
	Origin      string  `json:"origin"`
	Destination string  `json:"destination"`
	Bookings    int64   `json:"bookings"`
	Revenue     float64 `json:"revenue"`
	Share       float64 `json:"share"`
}

type CustomerSegment struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type IntegrationHealth struct {
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	Uptime  float64 `json:"uptime"`
	Latency int64   `json:"latency"`
}

type DashboardData struct {
	Summary            DashboardSummary    `json:"summary"`
	RevenueData        []RevenueData       `json:"revenueData"`
	BookingTrends      []BookingTrend      `json:"bookingTrends"`
	ConversionMetrics  []interface{}       `json:"conversionMetrics"`
	IntegrationHealth  []IntegrationHealth `json:"integrationHealth"`
	TopRoutes          []RoutePerformance  `json:"topRoutes"`
	CustomerSegments   []CustomerSegment   `json:"customerSegments"`
	PerformanceMetrics map[string]float64  `json:"performanceMetrics"`
}

type RealTimeMetrics struct {
	Revenue           float64            `json:"revenue"`
	Bookings          int64              `json:"bookings"`
	ActiveSessions    int64              `json:"activeSessions"`
	ConversionRate    float64            `json:"conversionRate"`
	LatencyP95        float64            `json:"latencyP95"`
	ErrorRate         float64            `json:"errorRate"`
	IntegrationStatus map[string]string  `json:"integrationStatus"`
}

type Alert struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	PartnerID   string    `json:"partnerId"`
}

type AnalyticsEngine struct {
	db *mongo.Database
}

func NewPartnerDashboardService(db *mongo.Database) *PartnerDashboardService {
	return &PartnerDashboardService{
		db:        db,
		wsClients: make(map[string]*websocket.Conn),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Configure properly for production
			},
		},
		analytics: &AnalyticsEngine{db: db},
		realTime:  &RealTimeMetrics{},
	}
}

func (pds *PartnerDashboardService) GetDashboardData(c *gin.Context) {
	partnerID := c.GetString("partnerID")
	timeRange := c.DefaultQuery("timeRange", "7d")
	
	if partnerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Partner ID required"})
		return
	}

	// Calculate time range
	endTime := time.Now()
	var startTime time.Time
	
	switch timeRange {
	case "1d":
		startTime = endTime.AddDate(0, 0, -1)
	case "7d":
		startTime = endTime.AddDate(0, 0, -7)
	case "30d":
		startTime = endTime.AddDate(0, 0, -30)
	case "90d":
		startTime = endTime.AddDate(0, 0, -90)
	case "1y":
		startTime = endTime.AddDate(-1, 0, 0)
	default:
		startTime = endTime.AddDate(0, 0, -7)
	}

	// Get dashboard data
	dashboardData, err := pds.analytics.GetDashboardData(partnerID, startTime, endTime)
	if err != nil {
		log.Printf("Error getting dashboard data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dashboard data"})
		return
	}

	c.JSON(http.StatusOK, dashboardData)
}

func (ae *AnalyticsEngine) GetDashboardData(partnerID string, startTime, endTime time.Time) (*DashboardData, error) {
	ctx := context.Background()
	
	// Get summary metrics
	summary, err := ae.getSummaryMetrics(ctx, partnerID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get revenue data
	revenueData, err := ae.getRevenueData(ctx, partnerID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get booking trends
	bookingTrends, err := ae.getBookingTrends(ctx, partnerID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get top routes
	topRoutes, err := ae.getTopRoutes(ctx, partnerID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get customer segments
	customerSegments, err := ae.getCustomerSegments(ctx, partnerID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Get integration health
	integrationHealth, err := ae.getIntegrationHealth(ctx, partnerID)
	if err != nil {
		return nil, err
	}

	// Get performance metrics
	performanceMetrics, err := ae.getPerformanceMetrics(ctx, partnerID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return &DashboardData{
		Summary:            *summary,
		RevenueData:        revenueData,
		BookingTrends:      bookingTrends,
		ConversionMetrics:  []interface{}{}, // TODO: Implement conversion metrics
		IntegrationHealth:  integrationHealth,
		TopRoutes:          topRoutes,
		CustomerSegments:   customerSegments,
		PerformanceMetrics: performanceMetrics,
	}, nil
}

func (ae *AnalyticsEngine) getSummaryMetrics(ctx context.Context, partnerID string, startTime, endTime time.Time) (*DashboardSummary, error) {
	bookingsCollection := ae.db.Collection("bookings")
	
	// Current period aggregation
	currentMatch := bson.M{
		"partnerId": partnerID,
		"createdAt": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
		"status": bson.M{"$in": []string{"confirmed", "completed"}},
	}

	currentPipeline := []bson.M{
		{"$match": currentMatch},
		{"$group": bson.M{
			"_id":           nil,
			"totalRevenue":  bson.M{"$sum": "$totalAmount"},
			"totalBookings": bson.M{"$sum": 1},
			"totalSearches": bson.M{"$sum": "$searchCount"},
		}},
	}

	cursor, err := bookingsCollection.Aggregate(ctx, currentPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var currentResults []bson.M
	if err = cursor.All(ctx, &currentResults); err != nil {
		return nil, err
	}

	var totalRevenue, totalBookings, totalSearches float64
	if len(currentResults) > 0 {
		result := currentResults[0]
		totalRevenue = result["totalRevenue"].(float64)
		totalBookings = result["totalBookings"].(float64)
		if searches, ok := result["totalSearches"].(float64); ok {
			totalSearches = searches
		}
	}

	// Previous period for comparison
	previousDuration := endTime.Sub(startTime)
	previousStartTime := startTime.Add(-previousDuration)
	previousEndTime := startTime

	previousMatch := bson.M{
		"partnerId": partnerID,
		"createdAt": bson.M{
			"$gte": previousStartTime,
			"$lte": previousEndTime,
		},
		"status": bson.M{"$in": []string{"confirmed", "completed"}},
	}

	previousPipeline := []bson.M{
		{"$match": previousMatch},
		{"$group": bson.M{
			"_id":             nil,
			"previousRevenue": bson.M{"$sum": "$totalAmount"},
			"previousBookings": bson.M{"$sum": 1},
		}},
	}

	prevCursor, err := bookingsCollection.Aggregate(ctx, previousPipeline)
	if err != nil {
		return nil, err
	}
	defer prevCursor.Close(ctx)

	var previousResults []bson.M
	if err = prevCursor.All(ctx, &previousResults); err != nil {
		return nil, err
	}

	var previousRevenue, previousBookings float64
	if len(previousResults) > 0 {
		result := previousResults[0]
		previousRevenue = result["previousRevenue"].(float64)
		previousBookings = result["previousBookings"].(float64)
	}

	// Calculate changes
	var revenueChange, bookingsChange float64
	if previousRevenue > 0 {
		revenueChange = ((totalRevenue - previousRevenue) / previousRevenue) * 100
	}
	if previousBookings > 0 {
		bookingsChange = ((totalBookings - previousBookings) / previousBookings) * 100
	}

	// Calculate conversion rate
	conversionRate := 0.0
	if totalSearches > 0 {
		conversionRate = (totalBookings / totalSearches) * 100
	}

	// Get health score from integration monitoring
	healthScore, err := ae.getHealthScore(ctx, partnerID)
	if err != nil {
		healthScore = 95.0 // Default value
	}

	return &DashboardSummary{
		TotalRevenue:     totalRevenue,
		TotalBookings:    int64(totalBookings),
		ConversionRate:   conversionRate,
		HealthScore:      healthScore,
		RevenueChange:    revenueChange,
		BookingsChange:   bookingsChange,
		ConversionChange: 0.0, // TODO: Calculate conversion change
	}, nil
}

func (ae *AnalyticsEngine) getRevenueData(ctx context.Context, partnerID string, startTime, endTime time.Time) ([]RevenueData, error) {
	collection := ae.db.Collection("bookings")
	
	pipeline := []bson.M{
		{"$match": bson.M{
			"partnerId": partnerID,
			"createdAt": bson.M{
				"$gte": startTime,
				"$lte": endTime,
			},
			"status": bson.M{"$in": []string{"confirmed", "completed"}},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$createdAt",
				},
			},
			"revenue": bson.M{"$sum": "$totalAmount"},
		}},
		{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []RevenueData
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		
		results = append(results, RevenueData{
			Date:    doc["_id"].(string),
			Revenue: doc["revenue"].(float64),
		})
	}

	return results, nil
}

func (ae *AnalyticsEngine) getBookingTrends(ctx context.Context, partnerID string, startTime, endTime time.Time) ([]BookingTrend, error) {
	collection := ae.db.Collection("search_events")
	
	pipeline := []bson.M{
		{"$match": bson.M{
			"partnerId": partnerID,
			"timestamp": bson.M{
				"$gte": startTime,
				"$lte": endTime,
			},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$timestamp",
				},
			},
			"searches": bson.M{"$sum": 1},
			"bookings": bson.M{"$sum": bson.M{"$cond": []interface{}{
				bson.M{"$eq": []string{"$eventType", "booking"}},
				1,
				0,
			}}},
		}},
		{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []BookingTrend
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		
		results = append(results, BookingTrend{
			Date:     doc["_id"].(string),
			Bookings: int64(doc["bookings"].(int32)),
			Searches: int64(doc["searches"].(int32)),
		})
	}

	return results, nil
}

func (ae *AnalyticsEngine) getTopRoutes(ctx context.Context, partnerID string, startTime, endTime time.Time) ([]RoutePerformance, error) {
	collection := ae.db.Collection("bookings")
	
	pipeline := []bson.M{
		{"$match": bson.M{
			"partnerId": partnerID,
			"createdAt": bson.M{
				"$gte": startTime,
				"$lte": endTime,
			},
			"status": bson.M{"$in": []string{"confirmed", "completed"}},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"origin":      "$origin",
				"destination": "$destination",
			},
			"bookings": bson.M{"$sum": 1},
			"revenue":  bson.M{"$sum": "$totalAmount"},
		}},
		{"$sort": bson.M{"revenue": -1}},
		{"$limit": 10},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []RoutePerformance
	totalRevenue := 0.0
	
	// First pass to calculate total
	var tempResults []bson.M
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		tempResults = append(tempResults, doc)
		totalRevenue += doc["revenue"].(float64)
	}

	// Second pass to calculate shares
	for _, doc := range tempResults {
		routeData := doc["_id"].(bson.M)
		revenue := doc["revenue"].(float64)
		share := 0.0
		if totalRevenue > 0 {
			share = (revenue / totalRevenue) * 100
		}
		
		results = append(results, RoutePerformance{
			Origin:      routeData["origin"].(string),
			Destination: routeData["destination"].(string),
			Bookings:    int64(doc["bookings"].(int32)),
			Revenue:     revenue,
			Share:       share,
		})
	}

	return results, nil
}

func (ae *AnalyticsEngine) getCustomerSegments(ctx context.Context, partnerID string, startTime, endTime time.Time) ([]CustomerSegment, error) {
	collection := ae.db.Collection("bookings")
	
	pipeline := []bson.M{
		{"$match": bson.M{
			"partnerId": partnerID,
			"createdAt": bson.M{
				"$gte": startTime,
				"$lte": endTime,
			},
			"status": bson.M{"$in": []string{"confirmed", "completed"}},
		}},
		{"$group": bson.M{
			"_id":      "$customerSegment",
			"count":    bson.M{"$sum": 1},
			"revenue":  bson.M{"$sum": "$totalAmount"},
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []CustomerSegment
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		
		segment := "Unknown"
		if seg, ok := doc["_id"].(string); ok && seg != "" {
			segment = seg
		}
		
		results = append(results, CustomerSegment{
			Name:  segment,
			Value: doc["revenue"].(float64),
		})
	}

	return results, nil
}

func (ae *AnalyticsEngine) getIntegrationHealth(ctx context.Context, partnerID string) ([]IntegrationHealth, error) {
	collection := ae.db.Collection("integration_monitoring")
	
	filter := bson.M{
		"partnerId": partnerID,
		"timestamp": bson.M{"$gte": time.Now().Add(-24 * time.Hour)},
	}

	pipeline := []bson.M{
		{"$match": filter},
		{"$group": bson.M{
			"_id":            "$integrationName",
			"totalRequests":  bson.M{"$sum": 1},
			"successCount":   bson.M{"$sum": bson.M{"$cond": []interface{}{
				bson.M{"$eq": []string{"$status", "success"}},
				1,
				0,
			}}},
			"avgLatency":     bson.M{"$avg": "$latency"},
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []IntegrationHealth
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		
		totalRequests := doc["totalRequests"].(int32)
		successCount := doc["successCount"].(int32)
		uptime := 0.0
		if totalRequests > 0 {
			uptime = (float64(successCount) / float64(totalRequests)) * 100
		}

		status := "healthy"
		if uptime < 95 {
			status = "critical"
		} else if uptime < 98 {
			status = "warning"
		}
		
		results = append(results, IntegrationHealth{
			Name:    doc["_id"].(string),
			Status:  status,
			Uptime:  uptime,
			Latency: int64(doc["avgLatency"].(float64)),
		})
	}

	return results, nil
}

func (ae *AnalyticsEngine) getPerformanceMetrics(ctx context.Context, partnerID string, startTime, endTime time.Time) (map[string]float64, error) {
	// This would typically aggregate performance data from various sources
	// For now, returning sample metrics
	return map[string]float64{
		"avgResponseTime":    95.5,
		"errorRate":         0.2,
		"throughputRPS":     145.7,
		"memoryUsage":       67.3,
		"cpuUsage":          34.8,
		"cacheHitRate":      89.2,
	}, nil
}

func (ae *AnalyticsEngine) getHealthScore(ctx context.Context, partnerID string) (float64, error) {
	// Calculate overall health score based on various metrics
	// This is a simplified calculation
	integrationHealth, err := ae.getIntegrationHealth(ctx, partnerID)
	if err != nil {
		return 95.0, nil // Default score
	}
	
	totalUptime := 0.0
	count := float64(len(integrationHealth))
	
	for _, health := range integrationHealth {
		totalUptime += health.Uptime
	}
	
	if count == 0 {
		return 95.0, nil
	}
	
	return totalUptime / count, nil
}

func (pds *PartnerDashboardService) HandleWebSocket(c *gin.Context) {
	partnerID := c.GetString("partnerID")
	if partnerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Partner ID required"})
		return
	}

	conn, err := pds.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Store connection
	pds.wsClients[partnerID] = conn

	// Send initial data
	dashboardData, err := pds.analytics.GetDashboardData(partnerID, time.Now().AddDate(0, 0, -7), time.Now())
	if err == nil {
		pds.sendWebSocketMessage(partnerID, "dashboard_update", dashboardData)
	}

	// Handle incoming messages
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// Handle subscription requests
		if msgType, ok := message["type"].(string); ok && msgType == "subscribe" {
			// Start real-time data streaming
			go pds.streamRealTimeData(partnerID)
		}
	}

	// Clean up connection
	delete(pds.wsClients, partnerID)
}

func (pds *PartnerDashboardService) streamRealTimeData(partnerID string) {
	ticker := time.NewTicker(10 * time.Second) // Update every 10 seconds
	defer ticker.Stop()

	for range ticker.C {
		if _, exists := pds.wsClients[partnerID]; !exists {
			break // Client disconnected
		}

		// Generate real-time metrics (in production, this would come from actual monitoring)
		metrics := &RealTimeMetrics{
			Revenue:        float64(time.Now().Unix() % 10000),
			Bookings:       time.Now().Unix() % 100,
			ActiveSessions: time.Now().Unix() % 500,
			ConversionRate: 2.5 + (float64(time.Now().Unix()%100) / 100),
			LatencyP95:     95.0 + (float64(time.Now().Unix()%50) / 10),
			ErrorRate:      0.1 + (float64(time.Now().Unix()%10) / 100),
			IntegrationStatus: map[string]string{
				"payment":    "healthy",
				"inventory":  "healthy",
				"messaging":  "warning",
			},
		}

		pds.sendWebSocketMessage(partnerID, "metrics_update", map[string]interface{}{
			"metrics": metrics,
		})
	}
}

func (pds *PartnerDashboardService) sendWebSocketMessage(partnerID, msgType string, data interface{}) {
	if conn, exists := pds.wsClients[partnerID]; exists {
		message := map[string]interface{}{
			"type": msgType,
			"data": data,
		}
		
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			conn.Close()
			delete(pds.wsClients, partnerID)
		}
	}
}

func (pds *PartnerDashboardService) ExportData(c *gin.Context) {
	partnerID := c.GetString("partnerID")
	format := c.DefaultQuery("format", "csv")
	timeRange := c.DefaultQuery("timeRange", "7d")
	
	if partnerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Partner ID required"})
		return
	}

	// Calculate time range
	endTime := time.Now()
	var startTime time.Time
	
	switch timeRange {
	case "1d":
		startTime = endTime.AddDate(0, 0, -1)
	case "7d":
		startTime = endTime.AddDate(0, 0, -7)
	case "30d":
		startTime = endTime.AddDate(0, 0, -30)
	case "90d":
		startTime = endTime.AddDate(0, 0, -90)
	case "1y":
		startTime = endTime.AddDate(-1, 0, 0)
	default:
		startTime = endTime.AddDate(0, 0, -7)
	}

	dashboardData, err := pds.analytics.GetDashboardData(partnerID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data for export"})
		return
	}

	switch format {
	case "csv":
		pds.exportCSV(c, dashboardData)
	case "xlsx":
		pds.exportExcel(c, dashboardData)
	case "pdf":
		pds.exportPDF(c, dashboardData)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported export format"})
	}
}

func (pds *PartnerDashboardService) exportCSV(c *gin.Context, data *DashboardData) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=partner-analytics.csv")
	
	// Create CSV content
	csvContent := "Date,Revenue,Bookings\n"
	for i, revenue := range data.RevenueData {
		bookings := int64(0)
		if i < len(data.BookingTrends) {
			bookings = data.BookingTrends[i].Bookings
		}
		csvContent += fmt.Sprintf("%s,%.2f,%d\n", revenue.Date, revenue.Revenue, bookings)
	}
	
	c.String(http.StatusOK, csvContent)
}

func (pds *PartnerDashboardService) exportExcel(c *gin.Context, data *DashboardData) {
	// TODO: Implement Excel export using excelize library
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Excel export not yet implemented"})
}

func (pds *PartnerDashboardService) exportPDF(c *gin.Context, data *DashboardData) {
	// TODO: Implement PDF export
	c.JSON(http.StatusNotImplemented, gin.H{"error": "PDF export not yet implemented"})
}

// RegisterRoutes registers all partner dashboard routes
func (pds *PartnerDashboardService) RegisterRoutes(router *gin.Engine) {
	partnerRoutes := router.Group("/api/v1/partners")
	{
		partnerRoutes.GET("/dashboard", pds.GetDashboardData)
		partnerRoutes.GET("/realtime", pds.HandleWebSocket)
		partnerRoutes.GET("/export", pds.ExportData)
	}
} 