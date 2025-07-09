package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/iaros/common/logging"
	"github.com/iaros/common/storage"
)

// SchemaRegistry manages API schemas and their versions centrally
type SchemaRegistry struct {
	schemas        map[string]*SchemaDefinition
	versions       map[string]map[string]*SchemaVersion
	compatibility  map[string]CompatibilityLevel
	storage        storage.Storage
	logger         logging.Logger
	mutex          sync.RWMutex
	
	// Events
	eventPublisher EventPublisher
	subscribers    []SchemaSubscriber
}

// SchemaDefinition represents a schema with metadata
type SchemaDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Subject     string                 `json:"subject"`
	Format      SchemaFormat           `json:"format"`
	Content     string                 `json:"content"`
	Version     string                 `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	
	// Compatibility
	Compatibility CompatibilityLevel    `json:"compatibility"`
	Dependencies  []string              `json:"dependencies"`
	
	// Usage tracking
	UsageCount    int64                 `json:"usage_count"`
	LastUsed      time.Time             `json:"last_used"`
}

// SchemaVersion represents a specific version of a schema
type SchemaVersion struct {
	Version       string                 `json:"version"`
	Content       string                 `json:"content"`
	Changes       []SchemaChange         `json:"changes"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	Deprecated    bool                   `json:"deprecated"`
	DeprecatedAt  *time.Time             `json:"deprecated_at,omitempty"`
	
	// Validation
	IsValid       bool                   `json:"is_valid"`
	ValidationErrors []ValidationError   `json:"validation_errors,omitempty"`
	
	// Migration info
	MigrationPath string                 `json:"migration_path,omitempty"`
	BackwardCompatible bool              `json:"backward_compatible"`
}

// SchemaChange represents a change between versions
type SchemaChange struct {
	Type        ChangeType             `json:"type"`
	Path        string                 `json:"path"`
	OldValue    interface{}            `json:"old_value,omitempty"`
	NewValue    interface{}            `json:"new_value,omitempty"`
	Description string                 `json:"description"`
	Breaking    bool                   `json:"breaking"`
}

// Enums and types
type SchemaFormat string
const (
	OpenAPIFormat   SchemaFormat = "openapi"
	JSONSchemaFormat SchemaFormat = "json-schema"
	AvroFormat      SchemaFormat = "avro"
	ProtobufFormat  SchemaFormat = "protobuf"
)

type CompatibilityLevel string
const (
	CompatibilityNone     CompatibilityLevel = "NONE"
	CompatibilityBackward CompatibilityLevel = "BACKWARD"
	CompatibilityForward  CompatibilityLevel = "FORWARD"
	CompatibilityFull     CompatibilityLevel = "FULL"
)

type ChangeType string
const (
	ChangeTypeAdded    ChangeType = "ADDED"
	ChangeTypeRemoved  ChangeType = "REMOVED"
	ChangeTypeModified ChangeType = "MODIFIED"
	ChangeTypeDeprecated ChangeType = "DEPRECATED"
)

// NewSchemaRegistry creates a new schema registry instance
func NewSchemaRegistry(storage storage.Storage, eventPublisher EventPublisher) *SchemaRegistry {
	return &SchemaRegistry{
		schemas:        make(map[string]*SchemaDefinition),
		versions:       make(map[string]map[string]*SchemaVersion),
		compatibility:  make(map[string]CompatibilityLevel),
		storage:        storage,
		logger:         logging.GetLogger("schema-registry"),
		eventPublisher: eventPublisher,
		subscribers:    make([]SchemaSubscriber, 0),
	}
}

// RegisterSchema registers a new schema or updates an existing one
func (sr *SchemaRegistry) RegisterSchema(ctx context.Context, schema *SchemaDefinition) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()
	
	sr.logger.Info("Registering schema", "id", schema.ID, "version", schema.Version)
	
	// Validate schema content
	if err := sr.validateSchemaContent(schema); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	
	// Check compatibility if updating existing schema
	if existing, exists := sr.schemas[schema.ID]; exists {
		if err := sr.checkCompatibility(existing, schema); err != nil {
			return fmt.Errorf("compatibility check failed: %w", err)
		}
	}
	
	// Create version entry
	if sr.versions[schema.ID] == nil {
		sr.versions[schema.ID] = make(map[string]*SchemaVersion)
	}
	
	// Calculate changes from previous version
	var changes []SchemaChange
	if len(sr.versions[schema.ID]) > 0 {
		changes = sr.calculateChanges(schema.ID, schema)
	}
	
	version := &SchemaVersion{
		Version:            schema.Version,
		Content:            schema.Content,
		Changes:            changes,
		CreatedAt:          time.Now(),
		CreatedBy:          schema.CreatedBy,
		IsValid:            true,
		BackwardCompatible: sr.isBackwardCompatible(changes),
	}
	
	// Store version
	sr.versions[schema.ID][schema.Version] = version
	
	// Update schema definition
	schema.UpdatedAt = time.Now()
	sr.schemas[schema.ID] = schema
	
	// Persist to storage
	if err := sr.persistSchema(ctx, schema); err != nil {
		return fmt.Errorf("failed to persist schema: %w", err)
	}
	
	// Publish event
	sr.publishSchemaEvent("schema.registered", schema)
	
	sr.logger.Info("Schema registered successfully", "id", schema.ID, "version", schema.Version)
	return nil
}

// GetSchema retrieves a schema by ID and version
func (sr *SchemaRegistry) GetSchema(ctx context.Context, id, version string) (*SchemaDefinition, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()
	
	schema, exists := sr.schemas[id]
	if !exists {
		return nil, fmt.Errorf("schema %s not found", id)
	}
	
	if version == "" || version == "latest" {
		// Return latest version
		schema.UsageCount++
		schema.LastUsed = time.Now()
		return schema, nil
	}
	
	// Return specific version
	versions, exists := sr.versions[id]
	if !exists {
		return nil, fmt.Errorf("no versions found for schema %s", id)
	}
	
	versionInfo, exists := versions[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found for schema %s", version, id)
	}
	
	if versionInfo.Deprecated {
		sr.logger.Warn("Using deprecated schema version", "id", id, "version", version)
	}
	
	// Create schema with specific version content
	versionedSchema := *schema
	versionedSchema.Content = versionInfo.Content
	versionedSchema.Version = version
	versionedSchema.UsageCount++
	versionedSchema.LastUsed = time.Now()
	
	return &versionedSchema, nil
}

// ListSchemas returns all schemas with optional filtering
func (sr *SchemaRegistry) ListSchemas(ctx context.Context, filters SchemaFilters) ([]*SchemaDefinition, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()
	
	var result []*SchemaDefinition
	
	for _, schema := range sr.schemas {
		if sr.matchesFilters(schema, filters) {
			result = append(result, schema)
		}
	}
	
	return result, nil
}

// GetSchemaVersions returns all versions of a schema
func (sr *SchemaRegistry) GetSchemaVersions(ctx context.Context, id string) ([]*SchemaVersion, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()
	
	versions, exists := sr.versions[id]
	if !exists {
		return nil, fmt.Errorf("schema %s not found", id)
	}
	
	var result []*SchemaVersion
	for _, version := range versions {
		result = append(result, version)
	}
	
	return result, nil
}

// DeprecateSchema marks a schema version as deprecated
func (sr *SchemaRegistry) DeprecateSchema(ctx context.Context, id, version string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()
	
	versions, exists := sr.versions[id]
	if !exists {
		return fmt.Errorf("schema %s not found", id)
	}
	
	versionInfo, exists := versions[version]
	if !exists {
		return fmt.Errorf("version %s not found for schema %s", version, id)
	}
	
	now := time.Now()
	versionInfo.Deprecated = true
	versionInfo.DeprecatedAt = &now
	
	// Publish deprecation event
	sr.publishSchemaEvent("schema.deprecated", sr.schemas[id])
	
	sr.logger.Info("Schema version deprecated", "id", id, "version", version)
	return nil
}

// ValidateAgainstSchema validates data against a specific schema
func (sr *SchemaRegistry) ValidateAgainstSchema(ctx context.Context, id, version string, data interface{}) error {
	schema, err := sr.GetSchema(ctx, id, version)
	if err != nil {
		return err
	}
	
	validator, err := sr.getValidator(schema.Format)
	if err != nil {
		return err
	}
	
	return validator.Validate(schema.Content, data)
}

// Helper methods

func (sr *SchemaRegistry) validateSchemaContent(schema *SchemaDefinition) error {
	validator, err := sr.getValidator(schema.Format)
	if err != nil {
		return err
	}
	
	return validator.ValidateSchema(schema.Content)
}

func (sr *SchemaRegistry) checkCompatibility(existing, new *SchemaDefinition) error {
	compatLevel := sr.compatibility[existing.ID]
	if compatLevel == CompatibilityNone {
		return nil // No compatibility checks required
	}
	
	checker, err := sr.getCompatibilityChecker(existing.Format)
	if err != nil {
		return err
	}
	
	return checker.CheckCompatibility(existing.Content, new.Content, compatLevel)
}

func (sr *SchemaRegistry) calculateChanges(schemaID string, newSchema *SchemaDefinition) []SchemaChange {
	// Get latest version for comparison
	versions := sr.versions[schemaID]
	if len(versions) == 0 {
		return []SchemaChange{}
	}
	
	// Find latest version
	var latestVersion *SchemaVersion
	for _, version := range versions {
		if latestVersion == nil || version.CreatedAt.After(latestVersion.CreatedAt) {
			latestVersion = version
		}
	}
	
	// Calculate diff
	differ, _ := sr.getDiffer(newSchema.Format)
	return differ.CalculateChanges(latestVersion.Content, newSchema.Content)
}

func (sr *SchemaRegistry) isBackwardCompatible(changes []SchemaChange) bool {
	for _, change := range changes {
		if change.Breaking {
			return false
		}
	}
	return true
}

func (sr *SchemaRegistry) persistSchema(ctx context.Context, schema *SchemaDefinition) error {
	data, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	
	key := fmt.Sprintf("schemas/%s", schema.ID)
	return sr.storage.Set(ctx, key, data)
}

func (sr *SchemaRegistry) publishSchemaEvent(eventType string, schema *SchemaDefinition) {
	if sr.eventPublisher != nil {
		event := SchemaEvent{
			Type:      eventType,
			SchemaID:  schema.ID,
			Version:   schema.Version,
			Timestamp: time.Now(),
			Schema:    schema,
		}
		sr.eventPublisher.Publish("schema.events", event)
	}
}

func (sr *SchemaRegistry) matchesFilters(schema *SchemaDefinition, filters SchemaFilters) bool {
	if filters.Format != "" && schema.Format != filters.Format {
		return false
	}
	if filters.Subject != "" && schema.Subject != filters.Subject {
		return false
	}
	if len(filters.Tags) > 0 {
		for _, tag := range filters.Tags {
			found := false
			for _, schemaTag := range schema.Tags {
				if schemaTag == tag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

// Interface definitions
type SchemaValidator interface {
	Validate(schema string, data interface{}) error
	ValidateSchema(schema string) error
}

type CompatibilityChecker interface {
	CheckCompatibility(oldSchema, newSchema string, level CompatibilityLevel) error
}

type SchemaDiffer interface {
	CalculateChanges(oldSchema, newSchema string) []SchemaChange
}

type EventPublisher interface {
	Publish(topic string, event interface{}) error
}

type SchemaSubscriber interface {
	OnSchemaEvent(event SchemaEvent) error
}

// Support types
type SchemaFilters struct {
	Format  SchemaFormat `json:"format,omitempty"`
	Subject string       `json:"subject,omitempty"`
	Tags    []string     `json:"tags,omitempty"`
}

type SchemaEvent struct {
	Type      string            `json:"type"`
	SchemaID  string            `json:"schema_id"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Schema    *SchemaDefinition `json:"schema"`
}

type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Factory methods for validators, checkers, and differs
func (sr *SchemaRegistry) getValidator(format SchemaFormat) (SchemaValidator, error) {
	switch format {
	case OpenAPIFormat:
		return &OpenAPIValidator{}, nil
	case JSONSchemaFormat:
		return &JSONSchemaValidator{}, nil
	case AvroFormat:
		return &AvroValidator{}, nil
	default:
		return nil, fmt.Errorf("unsupported schema format: %s", format)
	}
}

func (sr *SchemaRegistry) getCompatibilityChecker(format SchemaFormat) (CompatibilityChecker, error) {
	switch format {
	case OpenAPIFormat:
		return &OpenAPICompatibilityChecker{}, nil
	case JSONSchemaFormat:
		return &JSONSchemaCompatibilityChecker{}, nil
	case AvroFormat:
		return &AvroCompatibilityChecker{}, nil
	default:
		return nil, fmt.Errorf("unsupported schema format: %s", format)
	}
}

func (sr *SchemaRegistry) getDiffer(format SchemaFormat) (SchemaDiffer, error) {
	switch format {
	case OpenAPIFormat:
		return &OpenAPIDiffer{}, nil
	case JSONSchemaFormat:
		return &JSONSchemaDiffer{}, nil
	case AvroFormat:
		return &AvroDiffer{}, nil
	default:
		return nil, fmt.Errorf("unsupported schema format: %s", format)
	}
} 