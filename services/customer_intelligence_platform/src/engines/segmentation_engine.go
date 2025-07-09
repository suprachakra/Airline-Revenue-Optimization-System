package engines

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MLSegmentationEngine - Advanced ML-powered customer segmentation for customer intelligence
// Provides RFM analysis, behavioral clustering, and ML-based micro-segmentation
// for precise customer targeting and personalization
type MLSegmentationEngine struct {
	db                    *mongo.Database
	rfmAnalyzer          *RFMAnalyzer
	behavioralClusterer  *BehavioralClusterer
	featureStore         *FeatureStore
	segmentationRules    []SegmentationRule
	realTimeProcessor    *RealTimeProcessor
	modelManager         *ModelManager
}

// RFMAnalyzer - Recency, Frequency, Monetary analysis for airline customers
// Analyzes booking recency, travel frequency, and spending patterns
type RFMAnalyzer struct {
	db              *mongo.Database
	rfmThresholds   RFMThresholds
	scoringEngine   *RFMScoringEngine
	trendAnalyzer   *TrendAnalyzer
}

// BehavioralClusterer - ML clustering for behavioral customer segmentation
// Uses K-means, DBSCAN, and hierarchical clustering on behavioral features
type BehavioralClusterer struct {
	db               *mongo.Database
	clusteringModel  *ClusteringModel
	featureExtractor *BehavioralFeatureExtractor
	clusterProfiles  map[string]ClusterProfile
}

// FeatureStore - Customer feature engineering and storage for ML segmentation
// Manages feature computation, storage, and real-time updates
type FeatureStore struct {
	db              *mongo.Database
	featureRegistry map[string]FeatureDefinition
	featureCache    *FeatureCache
	realTimeFeatures *RealTimeFeatures
}

// CustomerSegment - Complete customer segment profile with ML-based classification
type CustomerSegment struct {
	ID                  primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	CustomerID          string                `bson:"customerId" json:"customerId"`
	GlobalCustomerID    string                `bson:"globalCustomerId" json:"globalCustomerId"`
	RFMSegment          RFMSegment            `bson:"rfmSegment" json:"rfmSegment"`
	BehavioralCluster   BehavioralCluster     `bson:"behavioralCluster" json:"behavioralCluster"`
	ValueSegment        string                `bson:"valueSegment" json:"valueSegment"`
	LifecycleStage      string                `bson:"lifecycleStage" json:"lifecycleStage"`
	TravelProfile       TravelProfile         `bson:"travelProfile" json:"travelProfile"`
	EngagementLevel     string                `bson:"engagementLevel" json:"engagementLevel"`
	ChurnRisk           ChurnRiskProfile      `bson:"churnRisk" json:"churnRisk"`
	Preferences         PreferenceProfile     `bson:"preferences" json:"preferences"`
	Features            map[string]float64    `bson:"features" json:"features"`
	SegmentationHistory []SegmentHistory      `bson:"segmentationHistory" json:"segmentationHistory"`
	LastUpdated         time.Time             `bson:"lastUpdated" json:"lastUpdated"`
	CreatedAt           time.Time             `bson:"createdAt" json:"createdAt"`
}

// RFMSegment - Detailed RFM analysis for airline customer value segmentation
type RFMSegment struct {
	RecencyScore        int       `bson:"recencyScore" json:"recencyScore"`         // 1-5 based on last booking
	FrequencyScore      int       `bson:"frequencyScore" json:"frequencyScore"`     // 1-5 based on booking frequency
	MonetaryScore       int       `bson:"monetaryScore" json:"monetaryScore"`       // 1-5 based on total spend
	RFMScore            string    `bson:"rfmScore" json:"rfmScore"`                 // Combined RFM score (e.g., "555")
	Segment             string    `bson:"segment" json:"segment"`                   // "Champions", "Loyal Customers", etc.
	LastBookingDate     time.Time `bson:"lastBookingDate" json:"lastBookingDate"`
	BookingFrequency    float64   `bson:"bookingFrequency" json:"bookingFrequency"` // Bookings per year
	TotalSpend          float64   `bson:"totalSpend" json:"totalSpend"`             // Total lifetime value
	AverageOrderValue   float64   `bson:"averageOrderValue" json:"averageOrderValue"`
	SegmentDescription  string    `bson:"segmentDescription" json:"segmentDescription"`
}

// BehavioralCluster - ML clustering results for behavioral segmentation
type BehavioralCluster struct {
	ClusterID           string             `bson:"clusterId" json:"clusterId"`
	ClusterName         string             `bson:"clusterName" json:"clusterName"`
	ClusterDescription  string             `bson:"clusterDescription" json:"clusterDescription"`
	Confidence          float64            `bson:"confidence" json:"confidence"`
	KeyBehaviors        []string           `bson:"keyBehaviors" json:"keyBehaviors"`
	BehavioralFeatures  map[string]float64 `bson:"behavioralFeatures" json:"behavioralFeatures"`
	ClusterCentroid     []float64          `bson:"clusterCentroid" json:"clusterCentroid"`
}

// TravelProfile - Detailed travel behavior and preference analysis
type TravelProfile struct {
	TravelFrequency     string    `bson:"travelFrequency" json:"travelFrequency"`     // "Low", "Medium", "High", "VeryHigh"
	PreferredClass      string    `bson:"preferredClass" json:"preferredClass"`       // "Economy", "Business", "First"
	PreferredRoutes     []string  `bson:"preferredRoutes" json:"preferredRoutes"`     // Top traveled routes
	SeasonalPatterns    []string  `bson:"seasonalPatterns" json:"seasonalPatterns"`   // Travel seasonality
	BookingLeadTime     int       `bson:"bookingLeadTime" json:"bookingLeadTime"`     // Average days before travel
	TripDuration        string    `bson:"tripDuration" json:"tripDuration"`           // "Short", "Medium", "Long"
	CompanionType       string    `bson:"companionType" json:"companionType"`         // "Solo", "Family", "Business"
	PurposeOfTravel     string    `bson:"purposeOfTravel" json:"purposeOfTravel"`     // "Business", "Leisure", "Mixed"
}

// ChurnRiskProfile - ML-based churn prediction and prevention
type ChurnRiskProfile struct {
	ChurnScore          float64   `bson:"churnScore" json:"churnScore"`               // 0-1 churn probability
	RiskLevel           string    `bson:"riskLevel" json:"riskLevel"`                 // "Low", "Medium", "High", "Critical"
	KeyRiskFactors      []string  `bson:"keyRiskFactors" json:"keyRiskFactors"`       // Factors driving churn risk
	PredictedChurnDate  *time.Time `bson:"predictedChurnDate" json:"predictedChurnDate"` // When churn might occur
	RetentionPriority   string    `bson:"retentionPriority" json:"retentionPriority"` // Retention campaign priority
	RecommendedActions  []string  `bson:"recommendedActions" json:"recommendedActions"` // Retention actions
}

// PreferenceProfile - Customer preferences for personalization
type PreferenceProfile struct {
	ChannelPreferences  []string           `bson:"channelPreferences" json:"channelPreferences"`   // Preferred booking channels
	CommunicationTiming string             `bson:"communicationTiming" json:"communicationTiming"` // When to communicate
	ContentPreferences  []string           `bson:"contentPreferences" json:"contentPreferences"`   // Content preferences
	OfferTypes          []string           `bson:"offerTypes" json:"offerTypes"`                   // Preferred offer types
	PriceSensitivity    string             `bson:"priceSensitivity" json:"priceSensitivity"`       // Price sensitivity level
	ServicePreferences  map[string]string  `bson:"servicePreferences" json:"servicePreferences"`   // Service preferences
}

// SegmentHistory - Historical segmentation changes for trend analysis
type SegmentHistory struct {
	Timestamp       time.Time `bson:"timestamp" json:"timestamp"`
	PreviousSegment string    `bson:"previousSegment" json:"previousSegment"`
	NewSegment      string    `bson:"newSegment" json:"newSegment"`
	Trigger         string    `bson:"trigger" json:"trigger"`         // What triggered the change
	Confidence      float64   `bson:"confidence" json:"confidence"`   // Confidence in new segment
}

// RFMThresholds - Configurable thresholds for RFM scoring
type RFMThresholds struct {
	RecencyThresholds   []int     `json:"recencyThresholds"`   // Days: [0-30, 31-90, 91-180, 181-365, 365+]
	FrequencyThresholds []int     `json:"frequencyThresholds"` // Bookings: [1, 2-3, 4-6, 7-10, 11+]
	MonetaryThresholds  []float64 `json:"monetaryThresholds"`  // Spend: [0-500, 501-1500, 1501-5000, 5001-15000, 15000+]
}

// SegmentationRule - Business rules for customer segmentation
type SegmentationRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"` // "rfm", "behavioral", "value", "lifecycle"
	Conditions      []SegmentCondition     `json:"conditions"`
	Actions         []SegmentAction        `json:"actions"`
	Priority        int                    `json:"priority"`
	Parameters      map[string]interface{} `json:"parameters"`
}

// SegmentCondition - Conditions for rule-based segmentation
type SegmentCondition struct {
	Field       string      `json:"field"`
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value"`
	Weight      float64     `json:"weight"`
}

// SegmentAction - Actions to take based on segmentation rules
type SegmentAction struct {
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// FeatureDefinition - Definition of features used in ML segmentation
type FeatureDefinition struct {
	Name            string                 `json:"name"`
	Type            string                 `json:"type"` // "numerical", "categorical", "boolean"
	Description     string                 `json:"description"`
	CalculationRule string                 `json:"calculationRule"`
	UpdateFrequency string                 `json:"updateFrequency"`
	Dependencies    []string               `json:"dependencies"`
	Parameters      map[string]interface{} `json:"parameters"`
}

// ClusterProfile - Profile of a behavioral cluster
type ClusterProfile struct {
	ClusterID       string             `json:"clusterId"`
	ClusterName     string             `json:"clusterName"`
	Description     string             `json:"description"`
	Centroid        []float64          `json:"centroid"`
	KeyFeatures     []string           `json:"keyFeatures"`
	CustomerCount   int                `json:"customerCount"`
	Characteristics map[string]string  `json:"characteristics"`
}

// ClusteringModel - ML clustering model configuration
type ClusteringModel struct {
	ModelID         string    `json:"modelId"`
	Algorithm       string    `json:"algorithm"` // "kmeans", "dbscan", "hierarchical"
	NumClusters     int       `json:"numClusters"`
	Features        []string  `json:"features"`
	LastTrained     time.Time `json:"lastTrained"`
	ModelAccuracy   float64   `json:"modelAccuracy"`
}

// Supporting types
type RFMScoringEngine struct{}
type TrendAnalyzer struct{}
type BehavioralFeatureExtractor struct{}
type FeatureCache struct{}
type RealTimeFeatures struct{}
type RealTimeProcessor struct{}
type ModelManager struct{}

// NewMLSegmentationEngine - Initialize the ML segmentation engine for customer intelligence
func NewMLSegmentationEngine(db *mongo.Database) *MLSegmentationEngine {
	// Configure RFM thresholds for airline industry
	rfmThresholds := RFMThresholds{
		RecencyThresholds:   []int{30, 90, 180, 365},         // Days since last booking
		FrequencyThresholds: []int{1, 3, 6, 10},              // Number of bookings per year
		MonetaryThresholds:  []float64{500, 1500, 5000, 15000}, // Total spend tiers
	}

	return &MLSegmentationEngine{
		db:                  db,
		rfmAnalyzer:        NewRFMAnalyzer(db, rfmThresholds),
		behavioralClusterer: NewBehavioralClusterer(db),
		featureStore:       NewFeatureStore(db),
		segmentationRules:  []SegmentationRule{},
		realTimeProcessor:  &RealTimeProcessor{},
		modelManager:       &ModelManager{},
	}
}

// NewRFMAnalyzer - Initialize RFM analyzer with airline-specific thresholds
func NewRFMAnalyzer(db *mongo.Database, thresholds RFMThresholds) *RFMAnalyzer {
	return &RFMAnalyzer{
		db:            db,
		rfmThresholds: thresholds,
		scoringEngine: &RFMScoringEngine{},
		trendAnalyzer: &TrendAnalyzer{},
	}
}

// NewBehavioralClusterer - Initialize behavioral clustering with ML models
func NewBehavioralClusterer(db *mongo.Database) *BehavioralClusterer {
	// Initialize with default cluster profiles for airline customers
	clusterProfiles := map[string]ClusterProfile{
		"business_frequent": {
			ClusterID:   "business_frequent",
			ClusterName: "Business Frequent Travelers",
			Description: "High-frequency business travelers with premium preferences",
			KeyFeatures: []string{"booking_frequency", "business_class_preference", "short_lead_time"},
		},
		"leisure_budget": {
			ClusterID:   "leisure_budget",
			ClusterName: "Budget Leisure Travelers",
			Description: "Price-sensitive leisure travelers with advance planning",
			KeyFeatures: []string{"price_sensitivity", "advance_booking", "seasonal_travel"},
		},
		"premium_occasional": {
			ClusterID:   "premium_occasional",
			ClusterName: "Premium Occasional Travelers",
			Description: "Low-frequency travelers who prefer premium services",
			KeyFeatures: []string{"premium_class", "high_spending", "low_frequency"},
		},
		"family_travelers": {
			ClusterID:   "family_travelers",
			ClusterName: "Family Travelers",
			Description: "Families traveling together with specific needs",
			KeyFeatures: []string{"group_booking", "family_services", "vacation_destinations"},
		},
	}

	return &BehavioralClusterer{
		db:               db,
		clusteringModel:  &ClusteringModel{Algorithm: "kmeans", NumClusters: 8},
		featureExtractor: &BehavioralFeatureExtractor{},
		clusterProfiles:  clusterProfiles,
	}
}

// NewFeatureStore - Initialize feature store for ML segmentation
func NewFeatureStore(db *mongo.Database) *FeatureStore {
	// Define core features for airline customer segmentation
	featureRegistry := map[string]FeatureDefinition{
		"booking_frequency": {
			Name:            "booking_frequency",
			Type:            "numerical",
			Description:     "Number of bookings per year",
			CalculationRule: "COUNT(bookings) WHERE booking_date >= DATE_SUB(NOW(), INTERVAL 1 YEAR)",
			UpdateFrequency: "daily",
		},
		"average_fare": {
			Name:            "average_fare",
			Type:            "numerical",
			Description:     "Average fare paid per booking",
			CalculationRule: "AVG(total_fare) WHERE booking_date >= DATE_SUB(NOW(), INTERVAL 1 YEAR)",
			UpdateFrequency: "daily",
		},
		"lead_time": {
			Name:            "lead_time",
			Type:            "numerical",
			Description:     "Average days between booking and travel",
			CalculationRule: "AVG(DATEDIFF(travel_date, booking_date))",
			UpdateFrequency: "weekly",
		},
		"destination_diversity": {
			Name:            "destination_diversity",
			Type:            "numerical",
			Description:     "Number of unique destinations visited",
			CalculationRule: "COUNT(DISTINCT destination)",
			UpdateFrequency: "monthly",
		},
	}

	return &FeatureStore{
		db:              db,
		featureRegistry: featureRegistry,
		featureCache:    &FeatureCache{},
		realTimeFeatures: &RealTimeFeatures{},
	}
}

// ProcessCustomerSegmentation - Main method to segment a customer using ML algorithms
func (mse *MLSegmentationEngine) ProcessCustomerSegmentation(customerID string) (*CustomerSegment, error) {
	startTime := time.Now()
	
	// Extract customer features from feature store
	features, err := mse.featureStore.ExtractCustomerFeatures(customerID)
	if err != nil {
		return nil, fmt.Errorf("feature extraction failed: %v", err)
	}
	
	// Perform RFM analysis
	rfmSegment, err := mse.rfmAnalyzer.AnalyzeCustomer(customerID, features)
	if err != nil {
		return nil, fmt.Errorf("RFM analysis failed: %v", err)
	}
	
	// Perform behavioral clustering
	behavioralCluster, err := mse.behavioralClusterer.ClusterCustomer(customerID, features)
	if err != nil {
		return nil, fmt.Errorf("behavioral clustering failed: %v", err)
	}
	
	// Determine value segment based on RFM and behavioral analysis
	valueSegment := mse.determineValueSegment(rfmSegment, behavioralCluster)
	
	// Determine lifecycle stage
	lifecycleStage := mse.determineLifecycleStage(features)
	
	// Determine engagement level
	engagementLevel := mse.determineEngagementLevel(features)
	
	// Analyze travel profile
	travelProfile := mse.analyzeTravelProfile(customerID, features)
	
	// Assess churn risk
	churnRisk := mse.assessChurnRisk(features, rfmSegment)
	
	// Analyze preferences
	preferences := mse.analyzePreferences(customerID, features)
	
	// Create comprehensive customer segment
	segment := &CustomerSegment{
		CustomerID:        customerID,
		GlobalCustomerID:  customerID, // Could be mapped to global ID if needed
		RFMSegment:        *rfmSegment,
		BehavioralCluster: *behavioralCluster,
		ValueSegment:      valueSegment,
		LifecycleStage:    lifecycleStage,
		TravelProfile:     *travelProfile,
		EngagementLevel:   engagementLevel,
		ChurnRisk:         *churnRisk,
		Preferences:       *preferences,
		Features:          features,
		LastUpdated:       time.Now(),
		CreatedAt:         time.Now(),
	}
	
	// Store segmentation results
	err = mse.storeCustomerSegment(segment)
	if err != nil {
		log.Printf("Failed to store customer segment: %v", err)
	}
	
	// Track performance
	processingTime := time.Since(startTime)
	log.Printf("Customer segmentation completed for %s in %v", customerID, processingTime)
	
	return segment, nil
}

// AnalyzeCustomer - Perform detailed RFM analysis for airline customer
func (ra *RFMAnalyzer) AnalyzeCustomer(customerID string, features map[string]float64) (*RFMSegment, error) {
	// Extract RFM metrics from features
	daysSinceLastBooking := int(features["days_since_last_booking"])
	bookingFrequency := int(features["booking_frequency"])
	totalSpend := features["total_spend"]
	averageOrderValue := features["average_order_value"]
	
	// Calculate RFM scores (1-5 scale)
	recencyScore := ra.calculateRecencyScore(daysSinceLastBooking)
	frequencyScore := ra.calculateFrequencyScore(bookingFrequency)
	monetaryScore := ra.calculateMonetaryScore(totalSpend)
	
	// Create RFM score string
	rfmScore := fmt.Sprintf("%d%d%d", recencyScore, frequencyScore, monetaryScore)
	
	// Determine segment based on RFM scores
	segment := ra.determineRFMSegment(recencyScore, frequencyScore, monetaryScore)
	description := ra.getRFMDescription(segment)
	
	return &RFMSegment{
		RecencyScore:        recencyScore,
		FrequencyScore:      frequencyScore,
		MonetaryScore:       monetaryScore,
		RFMScore:            rfmScore,
		Segment:             segment,
		LastBookingDate:     time.Now().AddDate(0, 0, -daysSinceLastBooking),
		BookingFrequency:    float64(bookingFrequency),
		TotalSpend:          totalSpend,
		AverageOrderValue:   averageOrderValue,
		SegmentDescription:  description,
	}, nil
}

// calculateRecencyScore - Score based on days since last booking
func (ra *RFMAnalyzer) calculateRecencyScore(recencyDays int) int {
	thresholds := ra.rfmThresholds.RecencyThresholds
	
	// Score 5 (most recent) to 1 (least recent)
	if recencyDays <= thresholds[0] {
		return 5 // 0-30 days
	} else if recencyDays <= thresholds[1] {
		return 4 // 31-90 days
	} else if recencyDays <= thresholds[2] {
		return 3 // 91-180 days
	} else if recencyDays <= thresholds[3] {
		return 2 // 181-365 days
	}
	return 1 // 365+ days
}

// calculateFrequencyScore - Score based on booking frequency
func (ra *RFMAnalyzer) calculateFrequencyScore(frequency int) int {
	thresholds := ra.rfmThresholds.FrequencyThresholds
	
	// Score 1 (lowest frequency) to 5 (highest frequency)
	if frequency <= thresholds[0] {
		return 1 // 1 booking
	} else if frequency <= thresholds[1] {
		return 2 // 2-3 bookings
	} else if frequency <= thresholds[2] {
		return 3 // 4-6 bookings
	} else if frequency <= thresholds[3] {
		return 4 // 7-10 bookings
	}
	return 5 // 11+ bookings
}

// calculateMonetaryScore - Score based on total spend
func (ra *RFMAnalyzer) calculateMonetaryScore(monetary float64) int {
	thresholds := ra.rfmThresholds.MonetaryThresholds
	
	// Score 1 (lowest spend) to 5 (highest spend)
	if monetary <= thresholds[0] {
		return 1 // 0-500
	} else if monetary <= thresholds[1] {
		return 2 // 501-1500
	} else if monetary <= thresholds[2] {
		return 3 // 1501-5000
	} else if monetary <= thresholds[3] {
		return 4 // 5001-15000
	}
	return 5 // 15000+
}

// determineRFMSegment - Map RFM scores to business segments
func (ra *RFMAnalyzer) determineRFMSegment(r, f, m int) string {
	// High-value segments
	if r >= 4 && f >= 4 && m >= 4 {
		return "Champions"
	} else if r >= 3 && f >= 3 && m >= 4 {
		return "Loyal Customers"
	} else if r >= 4 && f >= 1 && m >= 4 {
		return "Potential Loyalists"
	} else if r >= 4 && f >= 1 && m >= 1 {
		return "New Customers"
	} else if r >= 3 && f >= 2 && m >= 2 {
		return "Promising"
	} else if r >= 2 && f >= 2 && m >= 2 {
		return "Customers Needing Attention"
	} else if r >= 2 && f >= 1 && m >= 1 {
		return "About to Sleep"
	} else if r >= 1 && f >= 4 && m >= 4 {
		return "At Risk"
	} else if r >= 1 && f >= 1 && m >= 4 {
		return "Cannot Lose Them"
	} else if r >= 1 && f >= 2 && m >= 1 {
		return "Hibernating"
	}
	return "Lost"
}

// getRFMDescription - Get detailed description for RFM segment
func (ra *RFMAnalyzer) getRFMDescription(segment string) string {
	descriptions := map[string]string{
		"Champions":                   "Bought recently, buy often and spend the most. Reward them. Can be early adopters for new products.",
		"Loyal Customers":             "Spend good money with us often. Responsive to promotions.",
		"Potential Loyalists":         "Recent customers, but spent a good amount and bought more than once.",
		"New Customers":               "Bought most recently, but not often.",
		"Promising":                   "Recent shoppers, but haven't spent much.",
		"Customers Needing Attention": "Above average recency, frequency and monetary values. May not have bought very recently though.",
		"About to Sleep":              "Below average recency, frequency and monetary values. Will lose them if not reactivated.",
		"At Risk":                     "Spent big money and purchased often. But long time ago. Need to bring them back!",
		"Cannot Lose Them":            "Made biggest purchases, and often. But haven't returned for a long time.",
		"Hibernating":                 "Last purchase was long back, low spenders and low number of orders.",
		"Lost":                        "Lowest recency, frequency and monetary scores.",
	}
	
	if description, exists := descriptions[segment]; exists {
		return description
	}
	return "Unknown segment"
}

// Additional methods would continue here for behavioral clustering, feature extraction, etc.
// This provides the core RFM analysis and segmentation framework 