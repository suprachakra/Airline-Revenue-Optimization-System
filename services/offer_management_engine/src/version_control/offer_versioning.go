package version_control

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/iaros/common/logging"
	"github.com/iaros/common/storage"
	"github.com/iaros/services/offer_service/models"
)

// OfferVersionControl provides Git-like versioning for offers with audit trails
type OfferVersionControl struct {
	versions      map[string]*OfferRepository
	storage       storage.Storage
	logger        logging.Logger
	mutex         sync.RWMutex
	
	// Git-like features
	branches      map[string]*OfferBranch
	tags          map[string]*OfferTag
	mergeHistory  []*MergeRecord
	
	// Audit
	auditLogger   AuditLogger
	eventStream   EventStream
}

// OfferRepository represents a collection of offer versions (like a Git repo)
type OfferRepository struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Commits     []*OfferCommit            `json:"commits"`
	Head        string                    `json:"head"` // Current commit hash
	Branches    map[string]*OfferBranch   `json:"branches"`
	Tags        map[string]*OfferTag      `json:"tags"`
	
	// Metadata
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	CreatedBy   string                    `json:"created_by"`
	Owner       string                    `json:"owner"`
	Access      AccessLevel               `json:"access"`
	
	// Collaboration
	Contributors []string                 `json:"contributors"`
	Watchers    []string                  `json:"watchers"`
}

// OfferCommit represents a single version/commit of an offer
type OfferCommit struct {
	Hash        string                    `json:"hash"`
	Message     string                    `json:"message"`
	Author      string                    `json:"author"`
	Timestamp   time.Time                 `json:"timestamp"`
	Parent      string                    `json:"parent,omitempty"`
	Parents     []string                  `json:"parents,omitempty"` // For merges
	
	// Offer data
	OfferData   *models.Offer             `json:"offer_data"`
	Changes     []*OfferChange            `json:"changes"`
	
	// Validation
	IsValid     bool                      `json:"is_valid"`
	Validation  *ValidationResult         `json:"validation,omitempty"`
	
	// Deployment
	Deployed    bool                      `json:"deployed"`
	DeployedAt  *time.Time                `json:"deployed_at,omitempty"`
	Environment string                    `json:"environment,omitempty"`
}

// OfferBranch represents a development branch for offers
type OfferBranch struct {
	Name        string                    `json:"name"`
	Head        string                    `json:"head"` // Commit hash
	Protected   bool                      `json:"protected"`
	CreatedAt   time.Time                 `json:"created_at"`
	CreatedBy   string                    `json:"created_by"`
	
	// Branch policies
	RequiresPR  bool                      `json:"requires_pr"`
	MinReviews  int                       `json:"min_reviews"`
	AutoMerge   bool                      `json:"auto_merge"`
}

// OfferTag represents a tagged version (like a release)
type OfferTag struct {
	Name        string                    `json:"name"`
	Commit      string                    `json:"commit"`
	Message     string                    `json:"message"`
	CreatedAt   time.Time                 `json:"created_at"`
	CreatedBy   string                    `json:"created_by"`
	
	// Release info
	IsRelease   bool                      `json:"is_release"`
	ReleaseNotes string                   `json:"release_notes,omitempty"`
	Environment string                    `json:"environment,omitempty"`
}

// OfferChange represents a change between offer versions
type OfferChange struct {
	Type        ChangeType                `json:"type"`
	Path        string                    `json:"path"`
	Field       string                    `json:"field"`
	OldValue    interface{}               `json:"old_value,omitempty"`
	NewValue    interface{}               `json:"new_value"`
	Description string                    `json:"description"`
	Impact      ImpactLevel               `json:"impact"`
}

// MergeRecord tracks merge operations
type MergeRecord struct {
	ID          string                    `json:"id"`
	FromBranch  string                    `json:"from_branch"`
	ToBranch    string                    `json:"to_branch"`
	CommitHash  string                    `json:"commit_hash"`
	Author      string                    `json:"author"`
	Timestamp   time.Time                 `json:"timestamp"`
	Conflicts   []*MergeConflict          `json:"conflicts,omitempty"`
	Status      MergeStatus               `json:"status"`
}

// MergeConflict represents conflicts during merge
type MergeConflict struct {
	Path        string                    `json:"path"`
	Field       string                    `json:"field"`
	BaseValue   interface{}               `json:"base_value"`
	BranchValue interface{}               `json:"branch_value"`
	Resolved    bool                      `json:"resolved"`
	Resolution  interface{}               `json:"resolution,omitempty"`
}

// Enums
type AccessLevel string
const (
	AccessPublic  AccessLevel = "public"
	AccessPrivate AccessLevel = "private"
	AccessTeam    AccessLevel = "team"
)

type ChangeType string
const (
	ChangeAdded    ChangeType = "added"
	ChangeModified ChangeType = "modified"
	ChangeRemoved  ChangeType = "removed"
)

type ImpactLevel string
const (
	ImpactLow      ImpactLevel = "low"
	ImpactMedium   ImpactLevel = "medium"
	ImpactHigh     ImpactLevel = "high"
	ImpactCritical ImpactLevel = "critical"
)

type MergeStatus string
const (
	MergeStatusPending    MergeStatus = "pending"
	MergeStatusCompleted  MergeStatus = "completed"
	MergeStatusConflicted MergeStatus = "conflicted"
	MergeStatusFailed     MergeStatus = "failed"
)

// NewOfferVersionControl creates a new version control system
func NewOfferVersionControl(storage storage.Storage, auditLogger AuditLogger) *OfferVersionControl {
	return &OfferVersionControl{
		versions:     make(map[string]*OfferRepository),
		storage:      storage,
		logger:       logging.GetLogger("offer-version-control"),
		branches:     make(map[string]*OfferBranch),
		tags:         make(map[string]*OfferTag),
		mergeHistory: make([]*MergeRecord, 0),
		auditLogger:  auditLogger,
	}
}

// InitRepository initializes a new offer repository
func (ovc *OfferVersionControl) InitRepository(ctx context.Context, repoName, description, owner string) (*OfferRepository, error) {
	ovc.mutex.Lock()
	defer ovc.mutex.Unlock()
	
	repoID := fmt.Sprintf("repo_%s_%d", repoName, time.Now().Unix())
	
	repo := &OfferRepository{
		ID:           repoID,
		Name:         repoName,
		Description:  description,
		Commits:      make([]*OfferCommit, 0),
		Head:         "",
		Branches:     make(map[string]*OfferBranch),
		Tags:         make(map[string]*OfferTag),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    owner,
		Owner:        owner,
		Access:       AccessPrivate,
		Contributors: []string{owner},
		Watchers:     []string{owner},
	}
	
	// Create default main branch
	mainBranch := &OfferBranch{
		Name:       "main",
		Head:       "",
		Protected:  true,
		CreatedAt:  time.Now(),
		CreatedBy:  owner,
		RequiresPR: true,
		MinReviews: 1,
		AutoMerge:  false,
	}
	repo.Branches["main"] = mainBranch
	
	ovc.versions[repoID] = repo
	
	// Audit log
	ovc.auditLogger.LogAction(ctx, "repository.created", owner, map[string]interface{}{
		"repo_id":     repoID,
		"repo_name":   repoName,
		"description": description,
	})
	
	ovc.logger.Info("Repository initialized", "repo_id", repoID, "name", repoName, "owner", owner)
	return repo, nil
}

// Commit creates a new commit with the given offer
func (ovc *OfferVersionControl) Commit(ctx context.Context, repoID, branch, message, author string, offer *models.Offer) (*OfferCommit, error) {
	ovc.mutex.Lock()
	defer ovc.mutex.Unlock()
	
	repo, exists := ovc.versions[repoID]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoID)
	}
	
	branchInfo, exists := repo.Branches[branch]
	if !exists {
		return nil, fmt.Errorf("branch %s not found in repository %s", branch, repoID)
	}
	
	// Validate offer
	validation, err := ovc.validateOffer(offer)
	if err != nil {
		return nil, fmt.Errorf("offer validation failed: %w", err)
	}
	
	// Calculate changes from parent commit
	var changes []*OfferChange
	var parent string
	if branchInfo.Head != "" {
		parentCommit := ovc.getCommit(repo, branchInfo.Head)
		if parentCommit != nil {
			changes = ovc.calculateOfferChanges(parentCommit.OfferData, offer)
			parent = branchInfo.Head
		}
	}
	
	// Generate commit hash
	commitHash := ovc.generateCommitHash(repoID, branch, message, author, offer)
	
	commit := &OfferCommit{
		Hash:       commitHash,
		Message:    message,
		Author:     author,
		Timestamp:  time.Now(),
		Parent:     parent,
		OfferData:  offer,
		Changes:    changes,
		IsValid:    validation.IsValid,
		Validation: validation,
		Deployed:   false,
	}
	
	// Add commit to repository
	repo.Commits = append(repo.Commits, commit)
	branchInfo.Head = commitHash
	repo.Head = commitHash
	repo.UpdatedAt = time.Now()
	
	// Persist to storage
	if err := ovc.persistRepository(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to persist repository: %w", err)
	}
	
	// Audit log
	ovc.auditLogger.LogAction(ctx, "offer.committed", author, map[string]interface{}{
		"repo_id":     repoID,
		"branch":      branch,
		"commit_hash": commitHash,
		"message":     message,
		"changes":     len(changes),
	})
	
	ovc.logger.Info("Offer committed", "repo_id", repoID, "branch", branch, "commit", commitHash, "author", author)
	return commit, nil
}

// CreateBranch creates a new branch from the specified commit
func (ovc *OfferVersionControl) CreateBranch(ctx context.Context, repoID, branchName, fromCommit, author string) (*OfferBranch, error) {
	ovc.mutex.Lock()
	defer ovc.mutex.Unlock()
	
	repo, exists := ovc.versions[repoID]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoID)
	}
	
	if _, exists := repo.Branches[branchName]; exists {
		return nil, fmt.Errorf("branch %s already exists", branchName)
	}
	
	// Validate from commit exists
	if fromCommit == "" {
		fromCommit = repo.Head
	}
	if !ovc.commitExists(repo, fromCommit) {
		return nil, fmt.Errorf("commit %s not found", fromCommit)
	}
	
	branch := &OfferBranch{
		Name:       branchName,
		Head:       fromCommit,
		Protected:  false,
		CreatedAt:  time.Now(),
		CreatedBy:  author,
		RequiresPR: false,
		MinReviews: 0,
		AutoMerge:  false,
	}
	
	repo.Branches[branchName] = branch
	repo.UpdatedAt = time.Now()
	
	// Audit log
	ovc.auditLogger.LogAction(ctx, "branch.created", author, map[string]interface{}{
		"repo_id":     repoID,
		"branch_name": branchName,
		"from_commit": fromCommit,
	})
	
	ovc.logger.Info("Branch created", "repo_id", repoID, "branch", branchName, "from", fromCommit, "author", author)
	return branch, nil
}

// Merge merges one branch into another
func (ovc *OfferVersionControl) Merge(ctx context.Context, repoID, fromBranch, toBranch, author string) (*MergeRecord, error) {
	ovc.mutex.Lock()
	defer ovc.mutex.Unlock()
	
	repo, exists := ovc.versions[repoID]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoID)
	}
	
	fromBranchInfo, exists := repo.Branches[fromBranch]
	if !exists {
		return nil, fmt.Errorf("source branch %s not found", fromBranch)
	}
	
	toBranchInfo, exists := repo.Branches[toBranch]
	if !exists {
		return nil, fmt.Errorf("target branch %s not found", toBranch)
	}
	
	// Check if fast-forward merge is possible
	canFastForward := ovc.canFastForward(repo, fromBranchInfo.Head, toBranchInfo.Head)
	
	mergeID := fmt.Sprintf("merge_%d", time.Now().Unix())
	mergeRecord := &MergeRecord{
		ID:         mergeID,
		FromBranch: fromBranch,
		ToBranch:   toBranch,
		Author:     author,
		Timestamp:  time.Now(),
		Status:     MergeStatusPending,
	}
	
	if canFastForward {
		// Fast-forward merge
		toBranchInfo.Head = fromBranchInfo.Head
		mergeRecord.CommitHash = fromBranchInfo.Head
		mergeRecord.Status = MergeStatusCompleted
	} else {
		// Three-way merge
		conflicts, mergeCommit, err := ovc.performThreeWayMerge(ctx, repo, fromBranchInfo, toBranchInfo, author)
		if err != nil {
			mergeRecord.Status = MergeStatusFailed
			return mergeRecord, err
		}
		
		if len(conflicts) > 0 {
			mergeRecord.Conflicts = conflicts
			mergeRecord.Status = MergeStatusConflicted
		} else {
			toBranchInfo.Head = mergeCommit.Hash
			mergeRecord.CommitHash = mergeCommit.Hash
			mergeRecord.Status = MergeStatusCompleted
		}
	}
	
	ovc.mergeHistory = append(ovc.mergeHistory, mergeRecord)
	repo.UpdatedAt = time.Now()
	
	// Audit log
	ovc.auditLogger.LogAction(ctx, "branch.merged", author, map[string]interface{}{
		"repo_id":     repoID,
		"from_branch": fromBranch,
		"to_branch":   toBranch,
		"merge_id":    mergeID,
		"status":      mergeRecord.Status,
	})
	
	ovc.logger.Info("Branch merge attempted", "repo_id", repoID, "from", fromBranch, "to", toBranch, "status", mergeRecord.Status)
	return mergeRecord, nil
}

// CreateTag creates a new tag at the specified commit
func (ovc *OfferVersionControl) CreateTag(ctx context.Context, repoID, tagName, commit, message, author string, isRelease bool) (*OfferTag, error) {
	ovc.mutex.Lock()
	defer ovc.mutex.Unlock()
	
	repo, exists := ovc.versions[repoID]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoID)
	}
	
	if _, exists := repo.Tags[tagName]; exists {
		return nil, fmt.Errorf("tag %s already exists", tagName)
	}
	
	if commit == "" {
		commit = repo.Head
	}
	
	if !ovc.commitExists(repo, commit) {
		return nil, fmt.Errorf("commit %s not found", commit)
	}
	
	tag := &OfferTag{
		Name:      tagName,
		Commit:    commit,
		Message:   message,
		CreatedAt: time.Now(),
		CreatedBy: author,
		IsRelease: isRelease,
	}
	
	repo.Tags[tagName] = tag
	repo.UpdatedAt = time.Now()
	
	// Audit log
	ovc.auditLogger.LogAction(ctx, "tag.created", author, map[string]interface{}{
		"repo_id":    repoID,
		"tag_name":   tagName,
		"commit":     commit,
		"is_release": isRelease,
	})
	
	ovc.logger.Info("Tag created", "repo_id", repoID, "tag", tagName, "commit", commit, "author", author)
	return tag, nil
}

// GetCommitHistory returns the commit history for a branch
func (ovc *OfferVersionControl) GetCommitHistory(ctx context.Context, repoID, branch string, limit int) ([]*OfferCommit, error) {
	ovc.mutex.RLock()
	defer ovc.mutex.RUnlock()
	
	repo, exists := ovc.versions[repoID]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoID)
	}
	
	branchInfo, exists := repo.Branches[branch]
	if !exists {
		return nil, fmt.Errorf("branch %s not found", branch)
	}
	
	history := ovc.buildCommitHistory(repo, branchInfo.Head, limit)
	return history, nil
}

// GetDiff returns the differences between two commits
func (ovc *OfferVersionControl) GetDiff(ctx context.Context, repoID, fromCommit, toCommit string) ([]*OfferChange, error) {
	ovc.mutex.RLock()
	defer ovc.mutex.RUnlock()
	
	repo, exists := ovc.versions[repoID]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoID)
	}
	
	fromCommitData := ovc.getCommit(repo, fromCommit)
	toCommitData := ovc.getCommit(repo, toCommit)
	
	if fromCommitData == nil || toCommitData == nil {
		return nil, fmt.Errorf("one or both commits not found")
	}
	
	changes := ovc.calculateOfferChanges(fromCommitData.OfferData, toCommitData.OfferData)
	return changes, nil
}

// Helper methods

func (ovc *OfferVersionControl) validateOffer(offer *models.Offer) (*ValidationResult, error) {
	// Implement offer validation logic
	result := &ValidationResult{
		IsValid: true,
		Errors:  make([]ValidationError, 0),
	}
	
	// Basic validation checks
	if offer.ID == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "id",
			Message: "Offer ID is required",
			Code:    "MISSING_ID",
		})
	}
	
	if offer.Price <= 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "price",
			Message: "Offer price must be positive",
			Code:    "INVALID_PRICE",
		})
	}
	
	return result, nil
}

func (ovc *OfferVersionControl) calculateOfferChanges(oldOffer, newOffer *models.Offer) []*OfferChange {
	changes := make([]*OfferChange, 0)
	
	// Compare price
	if oldOffer.Price != newOffer.Price {
		changes = append(changes, &OfferChange{
			Type:        ChangeModified,
			Path:        "offer.price",
			Field:       "price",
			OldValue:    oldOffer.Price,
			NewValue:    newOffer.Price,
			Description: fmt.Sprintf("Price changed from %.2f to %.2f", oldOffer.Price, newOffer.Price),
			Impact:      ovc.determinePriceChangeImpact(oldOffer.Price, newOffer.Price),
		})
	}
	
	// Compare description
	if oldOffer.Description != newOffer.Description {
		changes = append(changes, &OfferChange{
			Type:        ChangeModified,
			Path:        "offer.description",
			Field:       "description",
			OldValue:    oldOffer.Description,
			NewValue:    newOffer.Description,
			Description: "Offer description updated",
			Impact:      ImpactLow,
		})
	}
	
	// Add more field comparisons as needed
	
	return changes
}

func (ovc *OfferVersionControl) determinePriceChangeImpact(oldPrice, newPrice float64) ImpactLevel {
	changePercent := ((newPrice - oldPrice) / oldPrice) * 100
	
	if changePercent > 20 || changePercent < -20 {
		return ImpactCritical
	} else if changePercent > 10 || changePercent < -10 {
		return ImpactHigh
	} else if changePercent > 5 || changePercent < -5 {
		return ImpactMedium
	}
	return ImpactLow
}

func (ovc *OfferVersionControl) generateCommitHash(repoID, branch, message, author string, offer *models.Offer) string {
	data := fmt.Sprintf("%s_%s_%s_%s_%s_%d", repoID, branch, message, author, offer.ID, time.Now().Unix())
	// In production, use proper hashing (SHA-256)
	return fmt.Sprintf("commit_%x", data)[:16]
}

func (ovc *OfferVersionControl) getCommit(repo *OfferRepository, hash string) *OfferCommit {
	for _, commit := range repo.Commits {
		if commit.Hash == hash {
			return commit
		}
	}
	return nil
}

func (ovc *OfferVersionControl) commitExists(repo *OfferRepository, hash string) bool {
	return ovc.getCommit(repo, hash) != nil
}

func (ovc *OfferVersionControl) canFastForward(repo *OfferRepository, fromCommit, toCommit string) bool {
	// Simple implementation - in production, would need proper graph traversal
	return ovc.isAncestor(repo, toCommit, fromCommit)
}

func (ovc *OfferVersionControl) isAncestor(repo *OfferRepository, ancestor, descendant string) bool {
	current := ovc.getCommit(repo, descendant)
	for current != nil {
		if current.Hash == ancestor {
			return true
		}
		if current.Parent == "" {
			break
		}
		current = ovc.getCommit(repo, current.Parent)
	}
	return false
}

func (ovc *OfferVersionControl) buildCommitHistory(repo *OfferRepository, startCommit string, limit int) []*OfferCommit {
	history := make([]*OfferCommit, 0)
	current := ovc.getCommit(repo, startCommit)
	count := 0
	
	for current != nil && (limit == 0 || count < limit) {
		history = append(history, current)
		count++
		
		if current.Parent == "" {
			break
		}
		current = ovc.getCommit(repo, current.Parent)
	}
	
	return history
}

func (ovc *OfferVersionControl) performThreeWayMerge(ctx context.Context, repo *OfferRepository, fromBranch, toBranch *OfferBranch, author string) ([]*MergeConflict, *OfferCommit, error) {
	// Simplified three-way merge implementation
	fromCommit := ovc.getCommit(repo, fromBranch.Head)
	toCommit := ovc.getCommit(repo, toBranch.Head)
	
	if fromCommit == nil || toCommit == nil {
		return nil, nil, fmt.Errorf("invalid commits for merge")
	}
	
	// Find common ancestor (simplified)
	baseCommit := ovc.findCommonAncestor(repo, fromCommit, toCommit)
	
	conflicts := make([]*MergeConflict, 0)
	mergedOffer := *toCommit.OfferData // Start with target branch
	
	// Simple merge logic - in production would be more sophisticated
	if fromCommit.OfferData.Price != toCommit.OfferData.Price {
		if baseCommit != nil && baseCommit.OfferData.Price != fromCommit.OfferData.Price && baseCommit.OfferData.Price != toCommit.OfferData.Price {
			// Conflict: both branches modified price
			conflicts = append(conflicts, &MergeConflict{
				Path:        "offer.price",
				Field:       "price",
				BaseValue:   baseCommit.OfferData.Price,
				BranchValue: fromCommit.OfferData.Price,
				Resolved:    false,
			})
		} else {
			// Use the newer price
			mergedOffer.Price = fromCommit.OfferData.Price
		}
	}
	
	if len(conflicts) > 0 {
		return conflicts, nil, nil
	}
	
	// Create merge commit
	commitHash := ovc.generateCommitHash(repo.ID, toBranch.Name, fmt.Sprintf("Merge %s into %s", fromBranch.Name, toBranch.Name), author, &mergedOffer)
	
	mergeCommit := &OfferCommit{
		Hash:      commitHash,
		Message:   fmt.Sprintf("Merge branch '%s' into '%s'", fromBranch.Name, toBranch.Name),
		Author:    author,
		Timestamp: time.Now(),
		Parents:   []string{toBranch.Head, fromBranch.Head},
		OfferData: &mergedOffer,
		Changes:   ovc.calculateOfferChanges(toCommit.OfferData, &mergedOffer),
		IsValid:   true,
	}
	
	repo.Commits = append(repo.Commits, mergeCommit)
	
	return nil, mergeCommit, nil
}

func (ovc *OfferVersionControl) findCommonAncestor(repo *OfferRepository, commit1, commit2 *OfferCommit) *OfferCommit {
	// Simplified common ancestor finding - in production would use proper algorithm
	ancestors1 := ovc.getAllAncestors(repo, commit1)
	current := commit2
	
	for current != nil {
		if _, exists := ancestors1[current.Hash]; exists {
			return current
		}
		if current.Parent == "" {
			break
		}
		current = ovc.getCommit(repo, current.Parent)
	}
	
	return nil
}

func (ovc *OfferVersionControl) getAllAncestors(repo *OfferRepository, commit *OfferCommit) map[string]*OfferCommit {
	ancestors := make(map[string]*OfferCommit)
	current := commit
	
	for current != nil {
		ancestors[current.Hash] = current
		if current.Parent == "" {
			break
		}
		current = ovc.getCommit(repo, current.Parent)
	}
	
	return ancestors
}

func (ovc *OfferVersionControl) persistRepository(ctx context.Context, repo *OfferRepository) error {
	data, err := json.Marshal(repo)
	if err != nil {
		return err
	}
	
	key := fmt.Sprintf("offer_repos/%s", repo.ID)
	return ovc.storage.Set(ctx, key, data)
}

// Interfaces and support types

type AuditLogger interface {
	LogAction(ctx context.Context, action, user string, details map[string]interface{}) error
}

type EventStream interface {
	Publish(event string, data interface{}) error
}

type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
} 