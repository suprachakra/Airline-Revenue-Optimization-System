package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/iaros/common/logging"
)

// AuthorizationManager handles RBAC and permission enforcement
type AuthorizationManager struct {
	policyEngine    *PolicyEngine
	roleManager     *RoleManager
	permissionCache *PermissionCache
	auditLogger     logging.Logger
	mu              sync.RWMutex
}

type PolicyEngine struct {
	policies map[string]*Policy
	rules    map[string]*AccessRule
	logger   logging.Logger
	mu       sync.RWMutex
}

type Policy struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Rules       []string          `json:"rules"`
	Priority    int               `json:"priority"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata"`
}

type AccessRule struct {
	ID          string            `json:"id"`
	Resource    string            `json:"resource"`
	Action      string            `json:"action"`
	Conditions  []Condition       `json:"conditions"`
	Effect      string            `json:"effect"` // allow, deny
	Priority    int               `json:"priority"`
	Metadata    map[string]string `json:"metadata"`
}

type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, in, not_in, gt, lt, contains
	Value    interface{} `json:"value"`
}

type RoleManager struct {
	roles       map[string]*Role
	hierarchies map[string][]string // role -> parent roles
	logger      logging.Logger
	mu          sync.RWMutex
}

type Role struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []string          `json:"permissions"`
	ParentRoles []string          `json:"parent_roles"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
	Metadata    map[string]string `json:"metadata"`
}

type PermissionCache struct {
	cache      map[string]*CachedPermissions
	expiration time.Duration
	logger     logging.Logger
	mu         sync.RWMutex
}

type CachedPermissions struct {
	UserID      string    `json:"user_id"`
	Permissions []string  `json:"permissions"`
	Roles       []string  `json:"roles"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type AuthorizationRequest struct {
	UserID     string            `json:"user_id"`
	Resource   string            `json:"resource"`
	Action     string            `json:"action"`
	Context    map[string]string `json:"context"`
	ClientIP   string            `json:"client_ip"`
	UserAgent  string            `json:"user_agent"`
	Timestamp  time.Time         `json:"timestamp"`
}

type AuthorizationResult struct {
	Allowed     bool              `json:"allowed"`
	Reason      string            `json:"reason"`
	PolicyID    string            `json:"policy_id"`
	RuleID      string            `json:"rule_id"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
}

func NewAuthorizationManager() *AuthorizationManager {
	return &AuthorizationManager{
		policyEngine:    NewPolicyEngine(),
		roleManager:     NewRoleManager(),
		permissionCache: NewPermissionCache(15 * time.Minute),
		auditLogger:     logging.GetLogger("authorization"),
	}
}

func NewPolicyEngine() *PolicyEngine {
	pe := &PolicyEngine{
		policies: make(map[string]*Policy),
		rules:    make(map[string]*AccessRule),
		logger:   logging.GetLogger("policy_engine"),
	}
	
	pe.loadDefaultPolicies()
	return pe
}

func NewRoleManager() *RoleManager {
	rm := &RoleManager{
		roles:       make(map[string]*Role),
		hierarchies: make(map[string][]string),
		logger:      logging.GetLogger("role_manager"),
	}
	
	rm.loadDefaultRoles()
	return rm
}

func NewPermissionCache(expiration time.Duration) *PermissionCache {
	return &PermissionCache{
		cache:      make(map[string]*CachedPermissions),
		expiration: expiration,
		logger:     logging.GetLogger("permission_cache"),
	}
}

// Authorization core logic
func (am *AuthorizationManager) Authorize(req *AuthorizationRequest) (*AuthorizationResult, error) {
	startTime := time.Now()
	defer func() {
		am.auditLogger.Debug("Authorization completed", 
			"user_id", req.UserID,
			"resource", req.Resource,
			"action", req.Action,
			"duration", time.Since(startTime))
	}()

	// Get user permissions from cache or compute
	permissions, err := am.getUserPermissions(req.UserID)
	if err != nil {
		return &AuthorizationResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Failed to get user permissions: %v", err),
		}, err
	}

	// Check direct permission match
	if am.hasDirectPermission(permissions, req.Resource, req.Action) {
		result := &AuthorizationResult{
			Allowed:     true,
			Reason:      "Direct permission granted",
			Permissions: permissions,
		}
		
		am.auditLogger.Info("Authorization granted - direct permission", 
			"user_id", req.UserID,
			"resource", req.Resource,
			"action", req.Action)
		
		return result, nil
	}

	// Evaluate policies and rules
	policyResult := am.policyEngine.EvaluatePolicies(req, permissions)
	if policyResult.Allowed {
		am.auditLogger.Info("Authorization granted - policy evaluation", 
			"user_id", req.UserID,
			"resource", req.Resource,
			"action", req.Action,
			"policy_id", policyResult.PolicyID)
		
		return policyResult, nil
	}

	// Authorization denied
	am.auditLogger.Warn("Authorization denied", 
		"user_id", req.UserID,
		"resource", req.Resource,
		"action", req.Action,
		"reason", policyResult.Reason)

	return &AuthorizationResult{
		Allowed: false,
		Reason:  "Access denied by policy",
	}, nil
}

func (am *AuthorizationManager) getUserPermissions(userID string) ([]string, error) {
	// Check cache first
	if cached := am.permissionCache.Get(userID); cached != nil {
		return cached.Permissions, nil
	}

	// Compute permissions from roles
	userRoles, err := am.getUserRoles(userID)
	if err != nil {
		return nil, err
	}

	permissions := am.roleManager.ComputePermissions(userRoles)
	
	// Cache the result
	am.permissionCache.Set(userID, permissions, userRoles)
	
	return permissions, nil
}

func (am *AuthorizationManager) getUserRoles(userID string) ([]string, error) {
	// In production, this would query the user database
	// For now, return default roles based on user type
	if userID == "admin" {
		return []string{"admin", "user"}, nil
	}
	return []string{"user"}, nil
}

func (am *AuthorizationManager) hasDirectPermission(permissions []string, resource, action string) bool {
	requiredPermission := fmt.Sprintf("%s:%s", resource, action)
	wildcardPermission := fmt.Sprintf("%s:*", resource)
	adminPermission := "*:*"

	for _, permission := range permissions {
		if permission == requiredPermission || 
		   permission == wildcardPermission || 
		   permission == adminPermission {
			return true
		}
	}
	return false
}

// Policy Engine implementation
func (pe *PolicyEngine) loadDefaultPolicies() {
	defaultPolicies := []*Policy{
		{
			ID:          "admin-full-access",
			Name:        "Administrator Full Access",
			Description: "Grants full access to administrators",
			Rules:       []string{"admin-rule"},
			Priority:    1,
			Active:      true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "user-read-access",
			Name:        "User Read Access",
			Description: "Grants read access to regular users",
			Rules:       []string{"user-read-rule"},
			Priority:    10,
			Active:      true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "pricing-analyst-access",
			Name:        "Pricing Analyst Access",
			Description: "Access for pricing analysts",
			Rules:       []string{"pricing-read-rule", "pricing-write-rule"},
			Priority:    5,
			Active:      true,
			CreatedAt:   time.Now(),
		},
	}

	defaultRules := []*AccessRule{
		{
			ID:       "admin-rule",
			Resource: "*",
			Action:   "*",
			Effect:   "allow",
			Priority: 1,
			Conditions: []Condition{
				{Field: "role", Operator: "in", Value: []string{"admin"}},
			},
		},
		{
			ID:       "user-read-rule",
			Resource: "*",
			Action:   "read",
			Effect:   "allow",
			Priority: 10,
			Conditions: []Condition{
				{Field: "role", Operator: "in", Value: []string{"user"}},
			},
		},
		{
			ID:       "pricing-read-rule",
			Resource: "pricing",
			Action:   "read",
			Effect:   "allow",
			Priority: 5,
			Conditions: []Condition{
				{Field: "role", Operator: "in", Value: []string{"pricing_analyst"}},
			},
		},
		{
			ID:       "pricing-write-rule",
			Resource: "pricing",
			Action:   "write",
			Effect:   "allow",
			Priority: 5,
			Conditions: []Condition{
				{Field: "role", Operator: "in", Value: []string{"pricing_analyst"}},
				{Field: "time", Operator: "in", Value: "business_hours"},
			},
		},
	}

	pe.mu.Lock()
	for _, policy := range defaultPolicies {
		pe.policies[policy.ID] = policy
	}
	for _, rule := range defaultRules {
		pe.rules[rule.ID] = rule
	}
	pe.mu.Unlock()

	pe.logger.Info("Loaded default policies and rules", 
		"policies", len(defaultPolicies), 
		"rules", len(defaultRules))
}

func (pe *PolicyEngine) EvaluatePolicies(req *AuthorizationRequest, userPermissions []string) *AuthorizationResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Get user roles from request context
	userRoles := pe.extractUserRoles(req)

	// Evaluate policies in priority order
	for _, policy := range pe.getSortedPolicies() {
		if !policy.Active {
			continue
		}

		// Evaluate all rules in the policy
		for _, ruleID := range policy.Rules {
			if rule, exists := pe.rules[ruleID]; exists {
				if pe.evaluateRule(rule, req, userRoles, userPermissions) {
					return &AuthorizationResult{
						Allowed:  rule.Effect == "allow",
						Reason:   fmt.Sprintf("Rule %s in policy %s", rule.ID, policy.ID),
						PolicyID: policy.ID,
						RuleID:   rule.ID,
						Permissions: userPermissions,
					}
				}
			}
		}
	}

	return &AuthorizationResult{
		Allowed: false,
		Reason:  "No matching policy found",
	}
}

func (pe *PolicyEngine) evaluateRule(rule *AccessRule, req *AuthorizationRequest, userRoles, userPermissions []string) bool {
	// Check resource and action match
	if !pe.matchesResource(rule.Resource, req.Resource) || 
	   !pe.matchesAction(rule.Action, req.Action) {
		return false
	}

	// Evaluate all conditions
	for _, condition := range rule.Conditions {
		if !pe.evaluateCondition(condition, req, userRoles, userPermissions) {
			return false
		}
	}

	return true
}

func (pe *PolicyEngine) evaluateCondition(condition Condition, req *AuthorizationRequest, userRoles, userPermissions []string) bool {
	switch condition.Field {
	case "role":
		return pe.evaluateRoleCondition(condition, userRoles)
	case "permission":
		return pe.evaluatePermissionCondition(condition, userPermissions)
	case "time":
		return pe.evaluateTimeCondition(condition, req.Timestamp)
	case "ip":
		return pe.evaluateIPCondition(condition, req.ClientIP)
	default:
		return false
	}
}

func (pe *PolicyEngine) evaluateRoleCondition(condition Condition, userRoles []string) bool {
	switch condition.Operator {
	case "in":
		if allowedRoles, ok := condition.Value.([]string); ok {
			for _, userRole := range userRoles {
				for _, allowedRole := range allowedRoles {
					if userRole == allowedRole {
						return true
					}
				}
			}
		}
	case "not_in":
		if deniedRoles, ok := condition.Value.([]string); ok {
			for _, userRole := range userRoles {
				for _, deniedRole := range deniedRoles {
					if userRole == deniedRole {
						return false
					}
				}
			}
			return true
		}
	}
	return false
}

func (pe *PolicyEngine) evaluatePermissionCondition(condition Condition, userPermissions []string) bool {
	switch condition.Operator {
	case "in":
		if requiredPermissions, ok := condition.Value.([]string); ok {
			for _, userPermission := range userPermissions {
				for _, requiredPermission := range requiredPermissions {
					if userPermission == requiredPermission {
						return true
					}
				}
			}
		}
	}
	return false
}

func (pe *PolicyEngine) evaluateTimeCondition(condition Condition, timestamp time.Time) bool {
	switch condition.Value {
	case "business_hours":
		hour := timestamp.Hour()
		return hour >= 9 && hour < 17
	case "after_hours":
		hour := timestamp.Hour()
		return hour < 9 || hour >= 17
	}
	return true
}

func (pe *PolicyEngine) evaluateIPCondition(condition Condition, clientIP string) bool {
	// Simplified IP range checking
	switch condition.Operator {
	case "in":
		if allowedIPs, ok := condition.Value.([]string); ok {
			for _, allowedIP := range allowedIPs {
				if clientIP == allowedIP {
					return true
				}
			}
		}
	}
	return false
}

func (pe *PolicyEngine) matchesResource(ruleResource, requestResource string) bool {
	return ruleResource == "*" || ruleResource == requestResource
}

func (pe *PolicyEngine) matchesAction(ruleAction, requestAction string) bool {
	return ruleAction == "*" || ruleAction == requestAction
}

func (pe *PolicyEngine) extractUserRoles(req *AuthorizationRequest) []string {
	if roles, exists := req.Context["roles"]; exists {
		// Parse roles from context
		return []string{roles} // Simplified
	}
	return []string{"user"} // Default role
}

func (pe *PolicyEngine) getSortedPolicies() []*Policy {
	policies := make([]*Policy, 0, len(pe.policies))
	for _, policy := range pe.policies {
		policies = append(policies, policy)
	}
	
	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(policies)-1; i++ {
		for j := i + 1; j < len(policies); j++ {
			if policies[i].Priority > policies[j].Priority {
				policies[i], policies[j] = policies[j], policies[i]
			}
		}
	}
	
	return policies
}

// Role Manager implementation
func (rm *RoleManager) loadDefaultRoles() {
	defaultRoles := []*Role{
		{
			ID:          "admin",
			Name:        "Administrator",
			Description: "System administrator with full access",
			Permissions: []string{"*:*"},
			Active:      true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "user",
			Name:        "Regular User",
			Description: "Regular user with limited access",
			Permissions: []string{"dashboard:read", "profile:read", "profile:write"},
			Active:      true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "pricing_analyst",
			Name:        "Pricing Analyst",
			Description: "Analyst with pricing access",
			Permissions: []string{
				"pricing:read", "pricing:write", "forecasting:read",
				"analytics:read", "reports:read", "reports:write",
			},
			ParentRoles: []string{"user"},
			Active:      true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "revenue_manager",
			Name:        "Revenue Manager",
			Description: "Manager with revenue optimization access",
			Permissions: []string{
				"revenue:read", "revenue:write", "pricing:read", "pricing:write",
				"forecasting:read", "forecasting:write", "analytics:read", "analytics:write",
				"offers:read", "offers:write", "reports:read", "reports:write",
			},
			ParentRoles: []string{"pricing_analyst"},
			Active:      true,
			CreatedAt:   time.Now(),
		},
	}

	rm.mu.Lock()
	for _, role := range defaultRoles {
		rm.roles[role.ID] = role
		if len(role.ParentRoles) > 0 {
			rm.hierarchies[role.ID] = role.ParentRoles
		}
	}
	rm.mu.Unlock()

	rm.logger.Info("Loaded default roles", "count", len(defaultRoles))
}

func (rm *RoleManager) ComputePermissions(userRoles []string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	permissionSet := make(map[string]bool)
	
	// Collect permissions from all roles including inherited ones
	for _, roleID := range userRoles {
		rm.collectRolePermissions(roleID, permissionSet, make(map[string]bool))
	}

	// Convert to slice
	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return permissions
}

func (rm *RoleManager) collectRolePermissions(roleID string, permissionSet map[string]bool, visited map[string]bool) {
	// Prevent infinite recursion
	if visited[roleID] {
		return
	}
	visited[roleID] = true

	// Get role
	role, exists := rm.roles[roleID]
	if !exists || !role.Active {
		return
	}

	// Add direct permissions
	for _, permission := range role.Permissions {
		permissionSet[permission] = true
	}

	// Add inherited permissions from parent roles
	for _, parentRoleID := range role.ParentRoles {
		rm.collectRolePermissions(parentRoleID, permissionSet, visited)
	}
}

// Permission Cache implementation
func (pc *PermissionCache) Get(userID string) *CachedPermissions {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if cached, exists := pc.cache[userID]; exists {
		if time.Now().Before(cached.ExpiresAt) {
			return cached
		}
		// Remove expired entry
		delete(pc.cache, userID)
	}
	return nil
}

func (pc *PermissionCache) Set(userID string, permissions, roles []string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cache[userID] = &CachedPermissions{
		UserID:      userID,
		Permissions: permissions,
		Roles:       roles,
		ExpiresAt:   time.Now().Add(pc.expiration),
	}
}

func (pc *PermissionCache) Invalidate(userID string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.cache, userID)
}

func (pc *PermissionCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.cache = make(map[string]*CachedPermissions)
}

// Management methods
func (am *AuthorizationManager) AddRole(role *Role) error {
	return am.roleManager.AddRole(role)
}

func (am *AuthorizationManager) AddPolicy(policy *Policy) error {
	return am.policyEngine.AddPolicy(policy)
}

func (am *AuthorizationManager) InvalidateUserPermissions(userID string) {
	am.permissionCache.Invalidate(userID)
}

func (rm *RoleManager) AddRole(role *Role) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.roles[role.ID]; exists {
		return fmt.Errorf("role %s already exists", role.ID)
	}

	role.CreatedAt = time.Now()
	rm.roles[role.ID] = role
	
	if len(role.ParentRoles) > 0 {
		rm.hierarchies[role.ID] = role.ParentRoles
	}

	rm.logger.Info("Role added", "role_id", role.ID, "name", role.Name)
	return nil
}

func (pe *PolicyEngine) AddPolicy(policy *Policy) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if _, exists := pe.policies[policy.ID]; exists {
		return fmt.Errorf("policy %s already exists", policy.ID)
	}

	policy.CreatedAt = time.Now()
	pe.policies[policy.ID] = policy

	pe.logger.Info("Policy added", "policy_id", policy.ID, "name", policy.Name)
	return nil
} 