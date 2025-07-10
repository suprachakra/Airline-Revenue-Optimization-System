package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/iaros/common/logging"
	"golang.org/x/crypto/pbkdf2"
)

// EncryptionManager handles all encryption/decryption operations and key management
type EncryptionManager struct {
	keyManager     *KeyManager
	dataEncryptor  *DataEncryptor
	fieldEncryptor *FieldEncryptor
	keyRotator     *KeyRotator
	logger         logging.Logger
	mu             sync.RWMutex
}

type KeyManager struct {
	masterKey    []byte
	dataKeys     map[string]*DataKey
	keyVersions  map[string]int
	hsm          *HSMClient
	logger       logging.Logger
	mu           sync.RWMutex
}

type DataKey struct {
	ID         string    `json:"id"`
	Key        []byte    `json:"key"`
	Version    int       `json:"version"`
	Purpose    string    `json:"purpose"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Active     bool      `json:"active"`
	Algorithm  string    `json:"algorithm"`
}

type DataEncryptor struct {
	keyManager *KeyManager
	logger     logging.Logger
}

type FieldEncryptor struct {
	keyManager *KeyManager
	logger     logging.Logger
}

type KeyRotator struct {
	keyManager *KeyManager
	config     *KeyRotationConfig
	logger     logging.Logger
	ticker     *time.Ticker
}

type KeyRotationConfig struct {
	Enabled          bool          `json:"enabled"`
	RotationInterval time.Duration `json:"rotation_interval"`
	GracePeriod      time.Duration `json:"grace_period"`
	AutoRotate       bool          `json:"auto_rotate"`
}

type HSMClient struct {
	endpoint string
	apiKey   string
	logger   logging.Logger
}

type EncryptionRequest struct {
	Data      []byte            `json:"data"`
	KeyID     string            `json:"key_id"`
	Purpose   string            `json:"purpose"`
	Context   map[string]string `json:"context"`
}

type EncryptionResult struct {
	EncryptedData []byte            `json:"encrypted_data"`
	KeyID         string            `json:"key_id"`
	KeyVersion    int               `json:"key_version"`
	Algorithm     string            `json:"algorithm"`
	IV            []byte            `json:"iv"`
	Context       map[string]string `json:"context"`
}

type DecryptionRequest struct {
	EncryptedData []byte            `json:"encrypted_data"`
	KeyID         string            `json:"key_id"`
	KeyVersion    int               `json:"key_version"`
	Algorithm     string            `json:"algorithm"`
	IV            []byte            `json:"iv"`
	Context       map[string]string `json:"context"`
}

func NewEncryptionManager(masterKey []byte) *EncryptionManager {
	keyManager := NewKeyManager(masterKey)
	
	return &EncryptionManager{
		keyManager:     keyManager,
		dataEncryptor:  NewDataEncryptor(keyManager),
		fieldEncryptor: NewFieldEncryptor(keyManager),
		keyRotator:     NewKeyRotator(keyManager, &KeyRotationConfig{
			Enabled:          true,
			RotationInterval: 24 * time.Hour,
			GracePeriod:      7 * 24 * time.Hour,
			AutoRotate:       true,
		}),
		logger: logging.GetLogger("encryption_manager"),
	}
}

func NewKeyManager(masterKey []byte) *KeyManager {
	km := &KeyManager{
		masterKey:   masterKey,
		dataKeys:    make(map[string]*DataKey),
		keyVersions: make(map[string]int),
		hsm:         NewHSMClient(),
		logger:      logging.GetLogger("key_manager"),
	}
	
	// Initialize default data keys
	km.initializeDefaultKeys()
	return km
}

func NewDataEncryptor(keyManager *KeyManager) *DataEncryptor {
	return &DataEncryptor{
		keyManager: keyManager,
		logger:     logging.GetLogger("data_encryptor"),
	}
}

func NewFieldEncryptor(keyManager *KeyManager) *FieldEncryptor {
	return &FieldEncryptor{
		keyManager: keyManager,
		logger:     logging.GetLogger("field_encryptor"),
	}
}

func NewKeyRotator(keyManager *KeyManager, config *KeyRotationConfig) *KeyRotator {
	return &KeyRotator{
		keyManager: keyManager,
		config:     config,
		logger:     logging.GetLogger("key_rotator"),
	}
}

func NewHSMClient() *HSMClient {
	return &HSMClient{
		endpoint: "https://hsm.iaros.com",
		apiKey:   "hsm-api-key",
		logger:   logging.GetLogger("hsm_client"),
	}
}

// Core encryption methods
func (em *EncryptionManager) EncryptData(req *EncryptionRequest) (*EncryptionResult, error) {
	return em.dataEncryptor.Encrypt(req)
}

func (em *EncryptionManager) DecryptData(req *DecryptionRequest) ([]byte, error) {
	return em.dataEncryptor.Decrypt(req)
}

func (em *EncryptionManager) EncryptField(fieldName string, value string, context map[string]string) (string, error) {
	return em.fieldEncryptor.EncryptField(fieldName, value, context)
}

func (em *EncryptionManager) DecryptField(fieldName string, encryptedValue string, context map[string]string) (string, error) {
	return em.fieldEncryptor.DecryptField(fieldName, encryptedValue, context)
}

// Data Encryptor implementation
func (de *DataEncryptor) Encrypt(req *EncryptionRequest) (*EncryptionResult, error) {
	// Get appropriate encryption key
	dataKey, err := de.keyManager.GetDataKey(req.KeyID, req.Purpose)
	if err != nil {
		return nil, fmt.Errorf("failed to get data key: %w", err)
	}

	// Generate random IV
	iv := make([]byte, 12) // 96-bit IV for GCM
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(dataKey.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Use GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Prepare additional authenticated data (AAD)
	aad := de.prepareAAD(req.Context)

	// Encrypt data
	encryptedData := gcm.Seal(nil, iv, req.Data, aad)

	result := &EncryptionResult{
		EncryptedData: encryptedData,
		KeyID:         dataKey.ID,
		KeyVersion:    dataKey.Version,
		Algorithm:     "AES-256-GCM",
		IV:            iv,
		Context:       req.Context,
	}

	de.logger.Debug("Data encrypted successfully", 
		"key_id", dataKey.ID, 
		"data_size", len(req.Data),
		"encrypted_size", len(encryptedData))

	return result, nil
}

func (de *DataEncryptor) Decrypt(req *DecryptionRequest) ([]byte, error) {
	// Get decryption key
	dataKey, err := de.keyManager.GetDataKeyByVersion(req.KeyID, req.KeyVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get data key: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(dataKey.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Use GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Prepare AAD
	aad := de.prepareAAD(req.Context)

	// Decrypt data
	plaintext, err := gcm.Open(nil, req.IV, req.EncryptedData, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	de.logger.Debug("Data decrypted successfully", 
		"key_id", dataKey.ID, 
		"encrypted_size", len(req.EncryptedData),
		"decrypted_size", len(plaintext))

	return plaintext, nil
}

func (de *DataEncryptor) prepareAAD(context map[string]string) []byte {
	// Create additional authenticated data from context
	var aad string
	for key, value := range context {
		aad += fmt.Sprintf("%s=%s;", key, value)
	}
	return []byte(aad)
}

// Field Encryptor implementation (for database fields)
func (fe *FieldEncryptor) EncryptField(fieldName string, value string, context map[string]string) (string, error) {
	// Use field-specific key
	keyID := fmt.Sprintf("field_%s", fieldName)
	
	req := &EncryptionRequest{
		Data:    []byte(value),
		KeyID:   keyID,
		Purpose: "field_encryption",
		Context: context,
	}

	result, err := fe.encryptWithFormat(req)
	if err != nil {
		return "", err
	}

	// Encode as base64 for database storage
	encoded := base64.StdEncoding.EncodeToString(result)
	
	fe.logger.Debug("Field encrypted", "field", fieldName, "original_size", len(value))
	return encoded, nil
}

func (fe *FieldEncryptor) DecryptField(fieldName string, encryptedValue string, context map[string]string) (string, error) {
	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted value: %w", err)
	}

	// Parse the encrypted format
	decryptReq, err := fe.parseEncryptedFormat(decoded)
	if err != nil {
		return "", err
	}

	decryptReq.Context = context

	plaintext, err := fe.decryptParsed(decryptReq)
	if err != nil {
		return "", err
	}

	fe.logger.Debug("Field decrypted", "field", fieldName)
	return string(plaintext), nil
}

func (fe *FieldEncryptor) encryptWithFormat(req *EncryptionRequest) ([]byte, error) {
	// Get data key
	dataKey, err := fe.keyManager.GetDataKey(req.KeyID, req.Purpose)
	if err != nil {
		return nil, err
	}

	// Generate IV
	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Encrypt
	block, err := aes.NewCipher(dataKey.Key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	aad := []byte(fmt.Sprintf("key_id=%s;purpose=%s", dataKey.ID, req.Purpose))
	encryptedData := gcm.Seal(nil, iv, req.Data, aad)

	// Create formatted output: version(1) + key_version(4) + iv(12) + encrypted_data
	formatted := make([]byte, 1+4+12+len(encryptedData))
	formatted[0] = 1 // Format version
	
	// Key version (4 bytes, big endian)
	formatted[1] = byte(dataKey.Version >> 24)
	formatted[2] = byte(dataKey.Version >> 16)
	formatted[3] = byte(dataKey.Version >> 8)
	formatted[4] = byte(dataKey.Version)
	
	// IV
	copy(formatted[5:17], iv)
	
	// Encrypted data
	copy(formatted[17:], encryptedData)

	return formatted, nil
}

func (fe *FieldEncryptor) parseEncryptedFormat(data []byte) (*DecryptionRequest, error) {
	if len(data) < 17 {
		return nil, fmt.Errorf("invalid encrypted data format")
	}

	formatVersion := data[0]
	if formatVersion != 1 {
		return nil, fmt.Errorf("unsupported format version: %d", formatVersion)
	}

	// Parse key version
	keyVersion := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])

	// Parse IV
	iv := make([]byte, 12)
	copy(iv, data[5:17])

	// Parse encrypted data
	encryptedData := make([]byte, len(data)-17)
	copy(encryptedData, data[17:])

	return &DecryptionRequest{
		EncryptedData: encryptedData,
		KeyVersion:    keyVersion,
		Algorithm:     "AES-256-GCM",
		IV:            iv,
	}, nil
}

func (fe *FieldEncryptor) decryptParsed(req *DecryptionRequest) ([]byte, error) {
	// This would need the key ID - simplified for now
	dataKey, err := fe.keyManager.GetDataKeyByVersion("default", req.KeyVersion)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(dataKey.Key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	aad := []byte(fmt.Sprintf("key_id=%s;purpose=field_encryption", dataKey.ID))
	return gcm.Open(nil, req.IV, req.EncryptedData, aad)
}

// Key Manager implementation
func (km *KeyManager) initializeDefaultKeys() {
	defaultKeys := []*DataKey{
		{
			ID:        "default",
			Key:       km.deriveKey("default", 1),
			Version:   1,
			Purpose:   "general",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
			Active:    true,
			Algorithm: "AES-256",
		},
		{
			ID:        "pii",
			Key:       km.deriveKey("pii", 1),
			Version:   1,
			Purpose:   "pii_encryption",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
			Active:    true,
			Algorithm: "AES-256",
		},
		{
			ID:        "pricing",
			Key:       km.deriveKey("pricing", 1),
			Version:   1,
			Purpose:   "pricing_data",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
			Active:    true,
			Algorithm: "AES-256",
		},
	}

	km.mu.Lock()
	for _, key := range defaultKeys {
		km.dataKeys[fmt.Sprintf("%s_v%d", key.ID, key.Version)] = key
		km.keyVersions[key.ID] = key.Version
	}
	km.mu.Unlock()

	km.logger.Info("Initialized default encryption keys", "count", len(defaultKeys))
}

func (km *KeyManager) deriveKey(keyID string, version int) []byte {
	// Use PBKDF2 to derive keys from master key
	salt := []byte(fmt.Sprintf("%s_v%d_salt", keyID, version))
	return pbkdf2.Key(km.masterKey, salt, 100000, 32, sha256.New)
}

func (km *KeyManager) GetDataKey(keyID, purpose string) (*DataKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	// Get latest version
	version, exists := km.keyVersions[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	keyName := fmt.Sprintf("%s_v%d", keyID, version)
	dataKey, exists := km.dataKeys[keyName]
	if !exists {
		return nil, fmt.Errorf("data key not found: %s", keyName)
	}

	if !dataKey.Active {
		return nil, fmt.Errorf("key is inactive: %s", keyName)
	}

	if time.Now().After(dataKey.ExpiresAt) {
		return nil, fmt.Errorf("key has expired: %s", keyName)
	}

	return dataKey, nil
}

func (km *KeyManager) GetDataKeyByVersion(keyID string, version int) (*DataKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keyName := fmt.Sprintf("%s_v%d", keyID, version)
	dataKey, exists := km.dataKeys[keyName]
	if !exists {
		return nil, fmt.Errorf("data key not found: %s", keyName)
	}

	return dataKey, nil
}

func (km *KeyManager) RotateKey(keyID string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	currentVersion := km.keyVersions[keyID]
	newVersion := currentVersion + 1

	newKey := &DataKey{
		ID:        keyID,
		Key:       km.deriveKey(keyID, newVersion),
		Version:   newVersion,
		Purpose:   km.getKeyPurpose(keyID),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
		Active:    true,
		Algorithm: "AES-256",
	}

	// Add new key
	keyName := fmt.Sprintf("%s_v%d", keyID, newVersion)
	km.dataKeys[keyName] = newKey
	km.keyVersions[keyID] = newVersion

	km.logger.Info("Key rotated", "key_id", keyID, "old_version", currentVersion, "new_version", newVersion)
	return nil
}

func (km *KeyManager) getKeyPurpose(keyID string) string {
	// Get purpose from existing key
	if currentVersion, exists := km.keyVersions[keyID]; exists {
		keyName := fmt.Sprintf("%s_v%d", keyID, currentVersion)
		if key, exists := km.dataKeys[keyName]; exists {
			return key.Purpose
		}
	}
	return "general"
}

// Key Rotator implementation
func (kr *KeyRotator) StartAutoRotation() {
	if !kr.config.Enabled || !kr.config.AutoRotate {
		return
	}

	kr.ticker = time.NewTicker(kr.config.RotationInterval)
	
	go func() {
		for range kr.ticker.C {
			kr.performRotation()
		}
	}()

	kr.logger.Info("Started automatic key rotation", "interval", kr.config.RotationInterval)
}

func (kr *KeyRotator) StopAutoRotation() {
	if kr.ticker != nil {
		kr.ticker.Stop()
		kr.logger.Info("Stopped automatic key rotation")
	}
}

func (kr *KeyRotator) performRotation() {
	keyIDs := []string{"default", "pii", "pricing"}
	
	for _, keyID := range keyIDs {
		if err := kr.keyManager.RotateKey(keyID); err != nil {
			kr.logger.Error("Failed to rotate key", "key_id", keyID, "error", err)
		}
	}
}

// HSM Client implementation (simplified)
func (hsm *HSMClient) GenerateKey(keyID string) ([]byte, error) {
	// In production, this would interact with actual HSM
	hsm.logger.Info("Generating key in HSM", "key_id", keyID)
	
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	
	return key, nil
}

func (hsm *HSMClient) EncryptWithHSM(keyID string, plaintext []byte) ([]byte, error) {
	// In production, this would use HSM for encryption
	hsm.logger.Debug("Encrypting with HSM", "key_id", keyID)
	return plaintext, nil // Placeholder
}

// Utility functions
func (em *EncryptionManager) GenerateMasterKey() []byte {
	masterKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		em.logger.Error("Failed to generate master key", "error", err)
		return nil
	}
	return masterKey
}

func (em *EncryptionManager) HashPassword(password string, salt []byte) string {
	if salt == nil {
		salt = make([]byte, 16)
		io.ReadFull(rand.Reader, salt)
	}
	
	hash := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash)
}

func (em *EncryptionManager) VerifyPassword(password, hashedPassword string) bool {
	parts := []string{}
	for _, part := range []string{hashedPassword} {
		if len(part) > 32 {
			parts = append(parts, part[:32], part[33:])
			break
		}
	}
	
	if len(parts) != 2 {
		return false
	}
	
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	
	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	
	actualHash := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)
	
	// Constant time comparison
	if len(expectedHash) != len(actualHash) {
		return false
	}
	
	var result byte
	for i := 0; i < len(expectedHash); i++ {
		result |= expectedHash[i] ^ actualHash[i]
	}
	
	return result == 0
}

// Key management API
func (em *EncryptionManager) RotateKey(keyID string) error {
	return em.keyManager.RotateKey(keyID)
}

func (em *EncryptionManager) StartKeyRotation() {
	em.keyRotator.StartAutoRotation()
}

func (em *EncryptionManager) StopKeyRotation() {
	em.keyRotator.StopAutoRotation()
} 