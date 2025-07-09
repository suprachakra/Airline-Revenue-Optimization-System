package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BiometricIntegration struct {
	db                *mongo.Database
	faceRecognition   *FaceRecognitionAPI
	documentVerifier  *DocumentVerificationAPI
	securityIntegration *SecurityIntegration
	boardingGates     *BoardingGateAPI
}

type BiometricProfile struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PassengerID    string            `bson:"passengerId" json:"passengerId"`
	FaceTemplate   string            `bson:"faceTemplate" json:"faceTemplate"`
	FingerprintTemplate string       `bson:"fingerprintTemplate" json:"fingerprintTemplate"`
	DocumentHash   string            `bson:"documentHash" json:"documentHash"`
	CreatedAt      time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time         `bson:"updatedAt" json:"updatedAt"`
	ExpiresAt      time.Time         `bson:"expiresAt" json:"expiresAt"`
	Status         string            `bson:"status" json:"status"`
}

type CheckInEvent struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PassengerID     string            `bson:"passengerId" json:"passengerId"`
	PNR             string            `bson:"pnr" json:"pnr"`
	FlightNumber    string            `bson:"flightNumber" json:"flightNumber"`
	Terminal        string            `bson:"terminal" json:"terminal"`
	KioskID         string            `bson:"kioskId" json:"kioskId"`
	Method          string            `bson:"method" json:"method"` // face, fingerprint, document
	VerificationScore float64         `bson:"verificationScore" json:"verificationScore"`
	Status          string            `bson:"status" json:"status"`
	DocumentsVerified []string        `bson:"documentsVerified" json:"documentsVerified"`
	SecurityCleared bool              `bson:"securityCleared" json:"securityCleared"`
	BoardingPassIssued bool           `bson:"boardingPassIssued" json:"boardingPassIssued"`
	Timestamp       time.Time         `bson:"timestamp" json:"timestamp"`
}

type BoardingEvent struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PassengerID     string            `bson:"passengerId" json:"passengerId"`
	PNR             string            `bson:"pnr" json:"pnr"`
	FlightNumber    string            `bson:"flightNumber" json:"flightNumber"`
	Gate            string            `bson:"gate" json:"gate"`
	SeatNumber      string            `bson:"seatNumber" json:"seatNumber"`
	Method          string            `bson:"method" json:"method"`
	VerificationScore float64         `bson:"verificationScore" json:"verificationScore"`
	BoardingGroup   string            `bson:"boardingGroup" json:"boardingGroup"`
	Status          string            `bson:"status" json:"status"`
	Timestamp       time.Time         `bson:"timestamp" json:"timestamp"`
}

type FaceRecognitionAPI struct {
	db *mongo.Database
}

type DocumentVerificationAPI struct {
	db *mongo.Database
}

type SecurityIntegration struct {
	db *mongo.Database
}

type BoardingGateAPI struct {
	db *mongo.Database
}

func NewBiometricIntegration(db *mongo.Database) *BiometricIntegration {
	return &BiometricIntegration{
		db:                  db,
		faceRecognition:     &FaceRecognitionAPI{db: db},
		documentVerifier:    &DocumentVerificationAPI{db: db},
		securityIntegration: &SecurityIntegration{db: db},
		boardingGates:       &BoardingGateAPI{db: db},
	}
}

func (bi *BiometricIntegration) EnrollBiometrics(c *gin.Context) {
	var enrollRequest struct {
		PassengerID     string `json:"passengerId" binding:"required"`
		FaceImage       string `json:"faceImage" binding:"required"`
		DocumentImage   string `json:"documentImage" binding:"required"`
		DocumentType    string `json:"documentType" binding:"required"`
		FingerprintData string `json:"fingerprintData"`
	}

	if err := c.ShouldBindJSON(&enrollRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify document authenticity
	docVerification, err := bi.documentVerifier.VerifyDocument(enrollRequest.DocumentImage, enrollRequest.DocumentType)
	if err != nil || !docVerification.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document verification failed"})
		return
	}

	// Extract face template
	faceTemplate, err := bi.faceRecognition.ExtractTemplate(enrollRequest.FaceImage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Face template extraction failed"})
		return
	}

	// Create document hash for privacy
	documentHash := bi.createDocumentHash(enrollRequest.DocumentImage)

	// Create biometric profile
	profile := BiometricProfile{
		ID:                  primitive.NewObjectID(),
		PassengerID:         enrollRequest.PassengerID,
		FaceTemplate:        faceTemplate,
		FingerprintTemplate: enrollRequest.FingerprintData,
		DocumentHash:        documentHash,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		ExpiresAt:           time.Now().AddDate(1, 0, 0), // 1 year expiry
		Status:              "active",
	}

	collection := bi.db.Collection("biometric_profiles")
	_, err = collection.InsertOne(context.Background(), profile)
	if err != nil {
		log.Printf("Error saving biometric profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save biometric profile"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":          "enrolled",
		"profileId":       profile.ID.Hex(),
		"verificationScore": docVerification.ConfidenceScore,
		"expiresAt":       profile.ExpiresAt,
	})
}

func (bi *BiometricIntegration) BiometricCheckIn(c *gin.Context) {
	var checkInRequest struct {
		PNR          string `json:"pnr" binding:"required"`
		FlightNumber string `json:"flightNumber" binding:"required"`
		Terminal     string `json:"terminal" binding:"required"`
		KioskID      string `json:"kioskId" binding:"required"`
		FaceImage    string `json:"faceImage"`
		Fingerprint  string `json:"fingerprint"`
		Method       string `json:"method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&checkInRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get passenger details from booking
	passenger, err := bi.getPassengerFromPNR(checkInRequest.PNR)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	// Get biometric profile
	profile, err := bi.getBiometricProfile(passenger.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Biometric profile not found. Please enroll first."})
		return
	}

	// Perform biometric verification
	var verificationScore float64
	switch checkInRequest.Method {
	case "face":
		if checkInRequest.FaceImage == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Face image required"})
			return
		}
		verificationScore, err = bi.faceRecognition.VerifyFace(checkInRequest.FaceImage, profile.FaceTemplate)
	case "fingerprint":
		if checkInRequest.Fingerprint == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Fingerprint data required"})
			return
		}
		verificationScore, err = bi.verifyFingerprint(checkInRequest.Fingerprint, profile.FingerprintTemplate)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification method"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Verification failed"})
		return
	}

	// Check verification threshold
	threshold := 0.85
	verified := verificationScore >= threshold

	// Security clearance check
	securityCleared := false
	if verified {
		securityCleared, _ = bi.securityIntegration.CheckSecurityStatus(passenger.ID, checkInRequest.FlightNumber)
	}

	// Create check-in event
	checkInEvent := CheckInEvent{
		ID:                 primitive.NewObjectID(),
		PassengerID:        passenger.ID,
		PNR:                checkInRequest.PNR,
		FlightNumber:       checkInRequest.FlightNumber,
		Terminal:           checkInRequest.Terminal,
		KioskID:            checkInRequest.KioskID,
		Method:             checkInRequest.Method,
		VerificationScore:  verificationScore,
		Status:             "success",
		SecurityCleared:    securityCleared,
		BoardingPassIssued: verified && securityCleared,
		Timestamp:          time.Now(),
	}

	if !verified {
		checkInEvent.Status = "failed"
	}

	collection := bi.db.Collection("checkin_events")
	_, err = collection.InsertOne(context.Background(), checkInEvent)
	if err != nil {
		log.Printf("Error saving check-in event: %v", err)
	}

	if !verified {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":            "verification_failed",
			"verificationScore": verificationScore,
			"threshold":         threshold,
			"message":           "Please try again or proceed to manual check-in",
		})
		return
	}

	// Generate boarding pass if successful
	var boardingPass map[string]interface{}
	if checkInEvent.BoardingPassIssued {
		boardingPass = bi.generateBoardingPass(passenger, checkInRequest.FlightNumber)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            "success",
		"verified":          verified,
		"verificationScore": verificationScore,
		"securityCleared":   securityCleared,
		"boardingPass":      boardingPass,
		"checkInTime":       checkInEvent.Timestamp,
		"nextSteps":         bi.getNextSteps(securityCleared),
	})
}

func (bi *BiometricIntegration) BiometricBoarding(c *gin.Context) {
	var boardingRequest struct {
		PNR          string `json:"pnr" binding:"required"`
		FlightNumber string `json:"flightNumber" binding:"required"`
		Gate         string `json:"gate" binding:"required"`
		FaceImage    string `json:"faceImage"`
		Method       string `json:"method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&boardingRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get passenger details
	passenger, err := bi.getPassengerFromPNR(boardingRequest.PNR)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	// Check if passenger has checked in
	checkedIn, err := bi.isPassengerCheckedIn(passenger.ID, boardingRequest.FlightNumber)
	if err != nil || !checkedIn {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Passenger must check in first"})
		return
	}

	// Get biometric profile
	profile, err := bi.getBiometricProfile(passenger.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Biometric profile not found"})
		return
	}

	// Verify biometrics
	verificationScore, err := bi.faceRecognition.VerifyFace(boardingRequest.FaceImage, profile.FaceTemplate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Verification failed"})
		return
	}

	threshold := 0.90 // Higher threshold for boarding
	verified := verificationScore >= threshold

	// Check boarding eligibility
	boardingAllowed := false
	boardingGroup := ""
	if verified {
		boardingAllowed, boardingGroup, _ = bi.boardingGates.CheckBoardingEligibility(
			passenger.ID, boardingRequest.FlightNumber, boardingRequest.Gate)
	}

	// Create boarding event
	boardingEvent := BoardingEvent{
		ID:                primitive.NewObjectID(),
		PassengerID:       passenger.ID,
		PNR:               boardingRequest.PNR,
		FlightNumber:      boardingRequest.FlightNumber,
		Gate:              boardingRequest.Gate,
		Method:            boardingRequest.Method,
		VerificationScore: verificationScore,
		BoardingGroup:     boardingGroup,
		Status:            "success",
		Timestamp:         time.Now(),
	}

	if !verified || !boardingAllowed {
		boardingEvent.Status = "denied"
	}

	collection := bi.db.Collection("boarding_events")
	_, err = collection.InsertOne(context.Background(), boardingEvent)
	if err != nil {
		log.Printf("Error saving boarding event: %v", err)
	}

	if !verified {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":            "verification_failed",
			"verificationScore": verificationScore,
			"threshold":         threshold,
		})
		return
	}

	if !boardingAllowed {
		c.JSON(http.StatusForbidden, gin.H{
			"status":        "boarding_denied",
			"reason":        "Boarding not allowed at this time",
			"boardingGroup": boardingGroup,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            "boarding_approved",
		"verificationScore": verificationScore,
		"boardingGroup":     boardingGroup,
		"gate":              boardingRequest.Gate,
		"boardingTime":      boardingEvent.Timestamp,
		"seatNumber":        passenger.SeatNumber,
	})
}

func (bi *BiometricIntegration) GetBiometricStatus(c *gin.Context) {
	passengerID := c.Param("passengerId")
	
	profile, err := bi.getBiometricProfile(passengerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Biometric profile not found"})
		return
	}

	// Get recent events
	checkInEvents, _ := bi.getRecentCheckInEvents(passengerID, 7)
	boardingEvents, _ := bi.getRecentBoardingEvents(passengerID, 7)

	c.JSON(http.StatusOK, gin.H{
		"profile":        profile,
		"checkInEvents":  checkInEvents,
		"boardingEvents": boardingEvents,
		"status":         profile.Status,
		"expiresAt":      profile.ExpiresAt,
	})
}

// Helper methods
func (bi *BiometricIntegration) createDocumentHash(documentImage string) string {
	hasher := sha256.New()
	hasher.Write([]byte(documentImage))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (bi *BiometricIntegration) getPassengerFromPNR(pnr string) (*Passenger, error) {
	// Mock passenger data - in production, this would query the booking system
	return &Passenger{
		ID:         "passenger_" + pnr,
		Name:       "John Doe",
		SeatNumber: "12A",
	}, nil
}

func (bi *BiometricIntegration) getBiometricProfile(passengerID string) (*BiometricProfile, error) {
	collection := bi.db.Collection("biometric_profiles")
	var profile BiometricProfile
	
	err := collection.FindOne(context.Background(), bson.M{
		"passengerId": passengerID,
		"status":      "active",
		"expiresAt":   bson.M{"$gt": time.Now()},
	}).Decode(&profile)
	
	return &profile, err
}

func (bi *BiometricIntegration) isPassengerCheckedIn(passengerID, flightNumber string) (bool, error) {
	collection := bi.db.Collection("checkin_events")
	
	count, err := collection.CountDocuments(context.Background(), bson.M{
		"passengerId":        passengerID,
		"flightNumber":       flightNumber,
		"status":             "success",
		"boardingPassIssued": true,
	})
	
	return count > 0, err
}

func (bi *BiometricIntegration) generateBoardingPass(passenger *Passenger, flightNumber string) map[string]interface{} {
	return map[string]interface{}{
		"passengerName": passenger.Name,
		"flightNumber":  flightNumber,
		"seatNumber":    passenger.SeatNumber,
		"gate":          "B12",
		"boardingTime":  time.Now().Add(90 * time.Minute),
		"qrCode":        fmt.Sprintf("BP_%s_%s", passenger.ID, flightNumber),
	}
}

func (bi *BiometricIntegration) getNextSteps(securityCleared bool) []string {
	if securityCleared {
		return []string{"Proceed to security", "Head to departure gate", "Board when called"}
	}
	return []string{"Complete security clearance", "Wait for boarding call"}
}

func (bi *BiometricIntegration) getRecentCheckInEvents(passengerID string, days int) ([]CheckInEvent, error) {
	collection := bi.db.Collection("checkin_events")
	cutoff := time.Now().AddDate(0, 0, -days)
	
	cursor, err := collection.Find(context.Background(), bson.M{
		"passengerId": passengerID,
		"timestamp":   bson.M{"$gte": cutoff},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var events []CheckInEvent
	cursor.All(context.Background(), &events)
	return events, nil
}

func (bi *BiometricIntegration) getRecentBoardingEvents(passengerID string, days int) ([]BoardingEvent, error) {
	collection := bi.db.Collection("boarding_events")
	cutoff := time.Now().AddDate(0, 0, -days)
	
	cursor, err := collection.Find(context.Background(), bson.M{
		"passengerId": passengerID,
		"timestamp":   bson.M{"$gte": cutoff},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var events []BoardingEvent
	cursor.All(context.Background(), &events)
	return events, nil
}

func (bi *BiometricIntegration) verifyFingerprint(provided, stored string) (float64, error) {
	// Mock fingerprint verification - in production, use proper biometric SDK
	if provided == stored {
		return 0.95, nil
	}
	return 0.60, nil
}

// API component methods
func (fr *FaceRecognitionAPI) ExtractTemplate(faceImage string) (string, error) {
	// Mock template extraction - in production, use proper face recognition SDK
	return fmt.Sprintf("TEMPLATE_%s", faceImage[:10]), nil
}

func (fr *FaceRecognitionAPI) VerifyFace(faceImage, template string) (float64, error) {
	// Mock face verification - in production, use proper face recognition SDK
	if len(faceImage) > 0 && len(template) > 0 {
		return 0.92, nil
	}
	return 0.65, nil
}

func (dv *DocumentVerificationAPI) VerifyDocument(documentImage, docType string) (*DocumentVerification, error) {
	// Mock document verification
	return &DocumentVerification{
		Valid:           true,
		ConfidenceScore: 0.95,
		DocumentType:    docType,
	}, nil
}

func (si *SecurityIntegration) CheckSecurityStatus(passengerID, flightNumber string) (bool, error) {
	// Mock security check - in production, integrate with TSA/security systems
	return true, nil
}

func (bg *BoardingGateAPI) CheckBoardingEligibility(passengerID, flightNumber, gate string) (bool, string, error) {
	// Mock boarding eligibility check
	return true, "Group 2", nil
}

// Supporting types
type Passenger struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SeatNumber string `json:"seatNumber"`
}

type DocumentVerification struct {
	Valid           bool    `json:"valid"`
	ConfidenceScore float64 `json:"confidenceScore"`
	DocumentType    string  `json:"documentType"`
}

// RegisterRoutes registers all biometric integration routes
func (bi *BiometricIntegration) RegisterRoutes(router *gin.Engine) {
	biometricRoutes := router.Group("/api/v1/biometric")
	{
		biometricRoutes.POST("/enroll", bi.EnrollBiometrics)
		biometricRoutes.POST("/checkin", bi.BiometricCheckIn)
		biometricRoutes.POST("/boarding", bi.BiometricBoarding)
		biometricRoutes.GET("/status/:passengerId", bi.GetBiometricStatus)
	}
} 