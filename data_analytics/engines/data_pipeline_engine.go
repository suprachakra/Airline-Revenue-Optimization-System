package engines

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DataPipelineEngine handles real-time data ingestion, transformation, and quality validation
type DataPipelineEngine struct {
	db            *sql.DB
	redis         *redis.Client
	kafkaProducer *kafka.Producer
	kafkaConsumer *kafka.Consumer
	config        *PipelineConfig
	processors    map[string]DataProcessor
	validators    map[string]DataValidator
	transformers  map[string]DataTransformer
	mutex         sync.RWMutex
	
	// Metrics
	recordsProcessed prometheus.Counter
	processingTime   prometheus.Histogram
	errorRate        prometheus.Gauge
	qualityScore     prometheus.Gauge
}

type PipelineConfig struct {
	KafkaBootstrapServers string
	RedisHost            string
	DatabaseURL          string
	BatchSize            int
	ProcessingInterval   time.Duration
	QualityThreshold     float64
	RetryAttempts        int
}

type DataRecord struct {
	ID          string                 `json:"id"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	QualityTags []string               `json:"quality_tags"`
}

type ProcessingResult struct {
	Success      bool                   `json:"success"`
	RecordID     string                 `json:"record_id"`
	Errors       []string               `json:"errors"`
	Warnings     []string               `json:"warnings"`
	QualityScore float64                `json:"quality_score"`
	Metrics      map[string]interface{} `json:"metrics"`
	ProcessedAt  time.Time              `json:"processed_at"`
}

type DataProcessor interface {
	Process(ctx context.Context, record *DataRecord) (*ProcessingResult, error)
	GetName() string
	GetConfig() map[string]interface{}
}

type DataValidator interface {
	Validate(ctx context.Context, record *DataRecord) (bool, []string, error)
	GetRules() []ValidationRule
}

type DataTransformer interface {
	Transform(ctx context.Context, record *DataRecord) (*DataRecord, error)
	GetTransformationType() string
}

type ValidationRule struct {
	Field      string `json:"field"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	Constraints map[string]interface{} `json:"constraints"`
}

// NewDataPipelineEngine creates a new data pipeline engine
func NewDataPipelineEngine(config *PipelineConfig) (*DataPipelineEngine, error) {
	// Initialize metrics
	recordsProcessed := promauto.NewCounter(prometheus.CounterOpts{
		Name: "iaros_pipeline_records_processed_total",
		Help: "Total number of records processed",
	})
	
	processingTime := promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "iaros_pipeline_processing_duration_seconds",
		Help: "Time taken to process records",
	})
	
	errorRate := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "iaros_pipeline_error_rate",
		Help: "Current error rate in pipeline",
	})
	
	qualityScore := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "iaros_pipeline_quality_score",
		Help: "Current data quality score",
	})

	// Initialize Kafka producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaBootstrapServers,
		"acks":             "all",
		"retries":          config.RetryAttempts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: config.RedisHost,
	})

	engine := &DataPipelineEngine{
		config:           config,
		kafkaProducer:    producer,
		redis:           rdb,
		processors:      make(map[string]DataProcessor),
		validators:      make(map[string]DataValidator),
		transformers:    make(map[string]DataTransformer),
		recordsProcessed: recordsProcessed,
		processingTime:   processingTime,
		errorRate:        errorRate,
		qualityScore:     qualityScore,
	}

	// Register built-in processors
	engine.registerProcessors()
	engine.registerValidators()
	engine.registerTransformers()

	return engine, nil
}

func (e *DataPipelineEngine) registerProcessors() {
	e.processors["booking_processor"] = NewBookingProcessor(e.db)
	e.processors["flight_processor"] = NewFlightProcessor(e.db)
	e.processors["revenue_processor"] = NewRevenueProcessor(e.db)
	e.processors["customer_processor"] = NewCustomerProcessor(e.db)
	e.processors["pricing_processor"] = NewPricingProcessor(e.db)
}

func (e *DataPipelineEngine) registerValidators() {
	e.validators["booking_validator"] = NewBookingValidator()
	e.validators["flight_validator"] = NewFlightValidator()
	e.validators["revenue_validator"] = NewRevenueValidator()
	e.validators["customer_validator"] = NewCustomerValidator()
}

func (e *DataPipelineEngine) registerTransformers() {
	e.transformers["normalization"] = NewNormalizationTransformer()
	e.transformers["enrichment"] = NewEnrichmentTransformer(e.redis)
	e.transformers["anonymization"] = NewAnonymizationTransformer()
	e.transformers["aggregation"] = NewAggregationTransformer()
}

// ProcessRecord processes a single data record through the pipeline
func (e *DataPipelineEngine) ProcessRecord(ctx context.Context, record *DataRecord) (*ProcessingResult, error) {
	startTime := time.Now()
	defer func() {
		e.processingTime.Observe(time.Since(startTime).Seconds())
		e.recordsProcessed.Inc()
	}()

	result := &ProcessingResult{
		RecordID:    record.ID,
		Success:     true,
		Errors:      []string{},
		Warnings:    []string{},
		ProcessedAt: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	// Step 1: Validation
	if err := e.validateRecord(ctx, record, result); err != nil {
		return result, err
	}

	// Step 2: Transformation
	transformedRecord, err := e.transformRecord(ctx, record)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("transformation error: %v", err))
		return result, nil
	}

	// Step 3: Processing
	if err := e.processRecord(ctx, transformedRecord, result); err != nil {
		return result, err
	}

	// Step 4: Quality Assessment
	result.QualityScore = e.calculateQualityScore(transformedRecord, result)
	e.qualityScore.Set(result.QualityScore)

	// Step 5: Store processed record
	if err := e.storeProcessedRecord(ctx, transformedRecord, result); err != nil {
		log.Printf("Failed to store processed record: %v", err)
		result.Warnings = append(result.Warnings, "storage warning")
	}

	return result, nil
}

func (e *DataPipelineEngine) validateRecord(ctx context.Context, record *DataRecord, result *ProcessingResult) error {
	validator, exists := e.validators[fmt.Sprintf("%s_validator", record.Type)]
	if !exists {
		result.Warnings = append(result.Warnings, "no specific validator found")
		return nil
	}

	valid, errors, err := validator.Validate(ctx, record)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !valid {
		result.Success = false
		result.Errors = append(result.Errors, errors...)
	}

	return nil
}

func (e *DataPipelineEngine) transformRecord(ctx context.Context, record *DataRecord) (*DataRecord, error) {
	transformedRecord := record

	// Apply transformations in sequence
	transformationOrder := []string{"normalization", "enrichment", "aggregation"}
	
	for _, transformerName := range transformationOrder {
		transformer, exists := e.transformers[transformerName]
		if !exists {
			continue
		}

		var err error
		transformedRecord, err = transformer.Transform(ctx, transformedRecord)
		if err != nil {
			return nil, fmt.Errorf("transformation %s failed: %w", transformerName, err)
		}
	}

	return transformedRecord, nil
}

func (e *DataPipelineEngine) processRecord(ctx context.Context, record *DataRecord, result *ProcessingResult) error {
	processor, exists := e.processors[fmt.Sprintf("%s_processor", record.Type)]
	if !exists {
		result.Warnings = append(result.Warnings, "no specific processor found")
		return nil
	}

	processResult, err := processor.Process(ctx, record)
	if err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	if !processResult.Success {
		result.Success = false
		result.Errors = append(result.Errors, processResult.Errors...)
	}

	result.Warnings = append(result.Warnings, processResult.Warnings...)
	
	// Merge metrics
	for key, value := range processResult.Metrics {
		result.Metrics[key] = value
	}

	return nil
}

func (e *DataPipelineEngine) calculateQualityScore(record *DataRecord, result *ProcessingResult) float64 {
	score := 1.0

	// Penalize for errors
	if len(result.Errors) > 0 {
		score -= 0.5
	}

	// Penalize for warnings
	score -= float64(len(result.Warnings)) * 0.1

	// Bonus for completeness
	completeness := e.calculateCompleteness(record)
	score = score * completeness

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

func (e *DataPipelineEngine) calculateCompleteness(record *DataRecord) float64 {
	requiredFields := e.getRequiredFields(record.Type)
	if len(requiredFields) == 0 {
		return 1.0
	}

	completedFields := 0
	for _, field := range requiredFields {
		if value, exists := record.Data[field]; exists && value != nil {
			completedFields++
		}
	}

	return float64(completedFields) / float64(len(requiredFields))
}

func (e *DataPipelineEngine) getRequiredFields(recordType string) []string {
	fieldMap := map[string][]string{
		"booking": {"booking_id", "passenger_id", "flight_id", "booking_date", "status"},
		"flight": {"flight_id", "route", "departure_time", "arrival_time", "aircraft_type"},
		"revenue": {"transaction_id", "amount", "currency", "booking_id", "payment_method"},
		"customer": {"customer_id", "name", "email", "registration_date"},
	}

	return fieldMap[recordType]
}

func (e *DataPipelineEngine) storeProcessedRecord(ctx context.Context, record *DataRecord, result *ProcessingResult) error {
	// Store in Redis for fast access
	recordData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	key := fmt.Sprintf("processed:%s:%s", record.Type, record.ID)
	err = e.redis.Set(ctx, key, recordData, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store in Redis: %w", err)
	}

	// Store result metadata
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	resultKey := fmt.Sprintf("result:%s:%s", record.Type, record.ID)
	err = e.redis.Set(ctx, resultKey, resultData, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store result in Redis: %w", err)
	}

	return nil
}

// StartRealTimeProcessing starts the real-time data processing pipeline
func (e *DataPipelineEngine) StartRealTimeProcessing(ctx context.Context) error {
	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": e.config.KafkaBootstrapServers,
		"group.id":          "iaros-pipeline",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	defer consumer.Close()

	// Subscribe to topics
	topics := []string{"bookings", "flights", "revenue", "customer-data"}
	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	log.Println("Started real-time data processing pipeline")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping data pipeline...")
			return nil
		default:
			msg, err := consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				continue // Timeout, continue polling
			}

			// Process message
			go func(message *kafka.Message) {
				var record DataRecord
				if err := json.Unmarshal(message.Value, &record); err != nil {
					log.Printf("Failed to unmarshal message: %v", err)
					return
				}

				result, err := e.ProcessRecord(ctx, &record)
				if err != nil {
					log.Printf("Failed to process record %s: %v", record.ID, err)
					e.errorRate.Inc()
					return
				}

				if !result.Success {
					log.Printf("Record %s processed with errors: %v", record.ID, result.Errors)
				}
			}(msg)
		}
	}
}

// Processor implementations (simplified)

type BookingProcessor struct {
	db *sql.DB
}

func NewBookingProcessor(db *sql.DB) *BookingProcessor {
	return &BookingProcessor{db: db}
}

func (p *BookingProcessor) Process(ctx context.Context, record *DataRecord) (*ProcessingResult, error) {
	// Implementation for booking data processing
	return &ProcessingResult{
		Success:      true,
		RecordID:     record.ID,
		QualityScore: 0.95,
		ProcessedAt:  time.Now(),
		Metrics: map[string]interface{}{
			"booking_value": record.Data["amount"],
			"route":        record.Data["route"],
		},
	}, nil
}

func (p *BookingProcessor) GetName() string {
	return "booking_processor"
}

func (p *BookingProcessor) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"type": "booking",
		"version": "1.0",
	}
}

// Similar implementations for other processors...

type FlightProcessor struct{ db *sql.DB }
func NewFlightProcessor(db *sql.DB) *FlightProcessor { return &FlightProcessor{db: db} }
func (p *FlightProcessor) Process(ctx context.Context, record *DataRecord) (*ProcessingResult, error) { return &ProcessingResult{Success: true, RecordID: record.ID}, nil }
func (p *FlightProcessor) GetName() string { return "flight_processor" }
func (p *FlightProcessor) GetConfig() map[string]interface{} { return map[string]interface{}{"type": "flight"} }

type RevenueProcessor struct{ db *sql.DB }
func NewRevenueProcessor(db *sql.DB) *RevenueProcessor { return &RevenueProcessor{db: db} }
func (p *RevenueProcessor) Process(ctx context.Context, record *DataRecord) (*ProcessingResult, error) { return &ProcessingResult{Success: true, RecordID: record.ID}, nil }
func (p *RevenueProcessor) GetName() string { return "revenue_processor" }
func (p *RevenueProcessor) GetConfig() map[string]interface{} { return map[string]interface{}{"type": "revenue"} }

type CustomerProcessor struct{ db *sql.DB }
func NewCustomerProcessor(db *sql.DB) *CustomerProcessor { return &CustomerProcessor{db: db} }
func (p *CustomerProcessor) Process(ctx context.Context, record *DataRecord) (*ProcessingResult, error) { return &ProcessingResult{Success: true, RecordID: record.ID}, nil }
func (p *CustomerProcessor) GetName() string { return "customer_processor" }
func (p *CustomerProcessor) GetConfig() map[string]interface{} { return map[string]interface{}{"type": "customer"} }

type PricingProcessor struct{ db *sql.DB }
func NewPricingProcessor(db *sql.DB) *PricingProcessor { return &PricingProcessor{db: db} }
func (p *PricingProcessor) Process(ctx context.Context, record *DataRecord) (*ProcessingResult, error) { return &ProcessingResult{Success: true, RecordID: record.ID}, nil }
func (p *PricingProcessor) GetName() string { return "pricing_processor" }
func (p *PricingProcessor) GetConfig() map[string]interface{} { return map[string]interface{}{"type": "pricing"} }

// Validator implementations (simplified)

type BookingValidator struct{}
func NewBookingValidator() *BookingValidator { return &BookingValidator{} }
func (v *BookingValidator) Validate(ctx context.Context, record *DataRecord) (bool, []string, error) { return true, nil, nil }
func (v *BookingValidator) GetRules() []ValidationRule { return []ValidationRule{} }

type FlightValidator struct{}
func NewFlightValidator() *FlightValidator { return &FlightValidator{} }
func (v *FlightValidator) Validate(ctx context.Context, record *DataRecord) (bool, []string, error) { return true, nil, nil }
func (v *FlightValidator) GetRules() []ValidationRule { return []ValidationRule{} }

type RevenueValidator struct{}
func NewRevenueValidator() *RevenueValidator { return &RevenueValidator{} }
func (v *RevenueValidator) Validate(ctx context.Context, record *DataRecord) (bool, []string, error) { return true, nil, nil }
func (v *RevenueValidator) GetRules() []ValidationRule { return []ValidationRule{} }

type CustomerValidator struct{}
func NewCustomerValidator() *CustomerValidator { return &CustomerValidator{} }
func (v *CustomerValidator) Validate(ctx context.Context, record *DataRecord) (bool, []string, error) { return true, nil, nil }
func (v *CustomerValidator) GetRules() []ValidationRule { return []ValidationRule{} }

// Transformer implementations (simplified)

type NormalizationTransformer struct{}
func NewNormalizationTransformer() *NormalizationTransformer { return &NormalizationTransformer{} }
func (t *NormalizationTransformer) Transform(ctx context.Context, record *DataRecord) (*DataRecord, error) { return record, nil }
func (t *NormalizationTransformer) GetTransformationType() string { return "normalization" }

type EnrichmentTransformer struct{ redis *redis.Client }
func NewEnrichmentTransformer(redis *redis.Client) *EnrichmentTransformer { return &EnrichmentTransformer{redis: redis} }
func (t *EnrichmentTransformer) Transform(ctx context.Context, record *DataRecord) (*DataRecord, error) { return record, nil }
func (t *EnrichmentTransformer) GetTransformationType() string { return "enrichment" }

type AnonymizationTransformer struct{}
func NewAnonymizationTransformer() *AnonymizationTransformer { return &AnonymizationTransformer{} }
func (t *AnonymizationTransformer) Transform(ctx context.Context, record *DataRecord) (*DataRecord, error) { return record, nil }
func (t *AnonymizationTransformer) GetTransformationType() string { return "anonymization" }

type AggregationTransformer struct{}
func NewAggregationTransformer() *AggregationTransformer { return &AggregationTransformer{} }
func (t *AggregationTransformer) Transform(ctx context.Context, record *DataRecord) (*DataRecord, error) { return record, nil }
func (t *AggregationTransformer) GetTransformationType() string { return "aggregation" } 