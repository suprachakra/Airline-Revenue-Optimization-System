package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

// RBACManager manages roles and permissions
type RBACManager struct {
	roles       map[string]*Role
	permissions map[string]*Permission
	mutex       sync.RWMutex
}

// Role represents a role with permissions
type Role struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Permissions []string `yaml:"permissions" json:"permissions"`
	Inherits    []string `yaml:"inherits" json:"inherits"`
}

// Permission represents a permission
type Permission struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Resource    string `yaml:"resource" json:"resource"`
	Action      string `yaml:"action" json:"action"`
}

// RBACConfig represents the RBAC configuration
type RBACConfig struct {
	Roles       []Role       `yaml:"roles" json:"roles"`
	Permissions []Permission `yaml:"permissions" json:"permissions"`
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(configPath string) (*RBACManager, error) {
	manager := &RBACManager{
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
	}

	// Load configuration
	if err := manager.loadConfig(configPath); err != nil {
		return nil, fmt.Errorf("failed to load RBAC config: %w", err)
	}

	return manager, nil
}

// loadConfig loads RBAC configuration from file
func (rm *RBACManager) loadConfig(configPath string) error {
	if configPath == "" {
		// Use default configuration
		rm.loadDefaultConfig()
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config RBACConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Load permissions
	for _, perm := range config.Permissions {
		rm.permissions[perm.Name] = &Permission{
			Name:        perm.Name,
			Description: perm.Description,
			Resource:    perm.Resource,
			Action:      perm.Action,
		}
	}

	// Load roles
	for _, role := range config.Roles {
		rm.roles[role.Name] = &Role{
			Name:        role.Name,
			Description: role.Description,
			Permissions: role.Permissions,
			Inherits:    role.Inherits,
		}
	}

	return nil
}

// loadDefaultConfig loads default RBAC configuration
func (rm *RBACManager) loadDefaultConfig() {
	// Default permissions
	permissions := []Permission{
		{Name: "read:pricing", Description: "Read pricing data", Resource: "pricing", Action: "read"},
		{Name: "write:pricing", Description: "Write pricing data", Resource: "pricing", Action: "write"},
		{Name: "read:forecasting", Description: "Read forecasting data", Resource: "forecasting", Action: "read"},
		{Name: "write:forecasting", Description: "Write forecasting data", Resource: "forecasting", Action: "write"},
		{Name: "read:offers", Description: "Read offers", Resource: "offers", Action: "read"},
		{Name: "write:offers", Description: "Write offers", Resource: "offers", Action: "write"},
		{Name: "read:orders", Description: "Read orders", Resource: "orders", Action: "read"},
		{Name: "write:orders", Description: "Write orders", Resource: "orders", Action: "write"},
		{Name: "read:users", Description: "Read user data", Resource: "users", Action: "read"},
		{Name: "write:users", Description: "Write user data", Resource: "users", Action: "write"},
		{Name: "admin:system", Description: "System administration", Resource: "system", Action: "admin"},
		{Name: "read:analytics", Description: "Read analytics data", Resource: "analytics", Action: "read"},
		{Name: "write:analytics", Description: "Write analytics data", Resource: "analytics", Action: "write"},
		{Name: "read:distribution", Description: "Read distribution data", Resource: "distribution", Action: "read"},
		{Name: "write:distribution", Description: "Write distribution data", Resource: "distribution", Action: "write"},
		{Name: "read:ancillary", Description: "Read ancillary services", Resource: "ancillary", Action: "read"},
		{Name: "write:ancillary", Description: "Write ancillary services", Resource: "ancillary", Action: "write"},
	}

	for _, perm := range permissions {
		rm.permissions[perm.Name] = &Permission{
			Name:        perm.Name,
			Description: perm.Description,
			Resource:    perm.Resource,
			Action:      perm.Action,
		}
	}

	// Default roles
	roles := []Role{
		{
			Name:        "admin",
			Description: "System administrator with full access",
			Permissions: []string{
				"admin:system",
				"read:pricing", "write:pricing",
				"read:forecasting", "write:forecasting",
				"read:offers", "write:offers",
				"read:orders", "write:orders",
				"read:users", "write:users",
				"read:analytics", "write:analytics",
				"read:distribution", "write:distribution",
				"read:ancillary", "write:ancillary",
			},
		},
		{
			Name:        "revenue_manager",
			Description: "Revenue management access",
			Permissions: []string{
				"read:pricing", "write:pricing",
				"read:forecasting", "write:forecasting",
				"read:offers", "write:offers",
				"read:analytics",
			},
		},
		{
			Name:        "sales_agent",
			Description: "Sales agent access",
			Permissions: []string{
				"read:pricing",
				"read:offers",
				"read:orders", "write:orders",
				"read:distribution",
				"read:ancillary", "write:ancillary",
			},
		},
		{
			Name:        "analyst",
			Description: "Data analyst access",
			Permissions: []string{
				"read:pricing",
				"read:forecasting",
				"read:offers",
				"read:orders",
				"read:analytics",
				"read:distribution",
				"read:ancillary",
			},
		},
		{
			Name:        "customer_service",
			Description: "Customer service access",
			Permissions: []string{
				"read:orders",
				"read:users",
				"read:offers",
				"read:ancillary",
			},
		},
		{
			Name:        "api_user",
			Description: "API access for external integrations",
			Permissions: []string{
				"read:pricing",
				"read:offers",
				"read:distribution",
				"read:ancillary",
			},
		},
		{
			Name:        "guest",
			Description: "Limited guest access",
			Permissions: []string{
				"read:offers",
			},
		},
	}

	for _, role := range roles {
		rm.roles[role.Name] = &Role{
			Name:        role.Name,
			Description: role.Description,
			Permissions: role.Permissions,
			Inherits:    role.Inherits,
		}
	}
}

// RoleHasPermission checks if a role has a specific permission
func (rm *RBACManager) RoleHasPermission(roleName, permission string) bool {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	role, exists := rm.roles[roleName]
	if !exists {
		return false
	}

	return rm.roleHasPermissionRecursive(role, permission, make(map[string]bool))
}

// roleHasPermissionRecursive checks permission recursively through role inheritance
func (rm *RBACManager) roleHasPermissionRecursive(role *Role, permission string, visited map[string]bool) bool {
	// Prevent infinite recursion
	if visited[role.Name] {
		return false
	}
	visited[role.Name] = true

	// Check direct permissions
	for _, perm := range role.Permissions {
		if perm == permission {
			return true
		}
	}

	// Check inherited roles
	for _, inheritedRoleName := range role.Inherits {
		if inheritedRole, exists := rm.roles[inheritedRoleName]; exists {
			if rm.roleHasPermissionRecursive(inheritedRole, permission, visited) {
				return true
			}
		}
	}

	return false
}

// UserHasPermission checks if a user has a specific permission
func (rm *RBACManager) UserHasPermission(userRoles []string, permission string) bool {
	for _, roleName := range userRoles {
		if rm.RoleHasPermission(roleName, permission) {
			return true
		}
	}
	return false
}

// GetRole returns a role by name
func (rm *RBACManager) GetRole(name string) (*Role, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	role, exists := rm.roles[name]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", name)
	}

	return role, nil
}

// GetPermission returns a permission by name
func (rm *RBACManager) GetPermission(name string) (*Permission, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	permission, exists := rm.permissions[name]
	if !exists {
		return nil, fmt.Errorf("permission not found: %s", name)
	}

	return permission, nil
}

// GetAllRoles returns all roles
func (rm *RBACManager) GetAllRoles() map[string]*Role {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	roles := make(map[string]*Role)
	for name, role := range rm.roles {
		roles[name] = role
	}

	return roles
}

// GetAllPermissions returns all permissions
func (rm *RBACManager) GetAllPermissions() map[string]*Permission {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	permissions := make(map[string]*Permission)
	for name, permission := range rm.permissions {
		permissions[name] = permission
	}

	return permissions
}

// GetRolePermissions returns all permissions for a role (including inherited)
func (rm *RBACManager) GetRolePermissions(roleName string) ([]string, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	role, exists := rm.roles[roleName]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", roleName)
	}

	permissions := make(map[string]bool)
	rm.collectRolePermissions(role, permissions, make(map[string]bool))

	result := make([]string, 0, len(permissions))
	for perm := range permissions {
		result = append(result, perm)
	}

	return result, nil
}

// collectRolePermissions recursively collects all permissions for a role
func (rm *RBACManager) collectRolePermissions(role *Role, permissions map[string]bool, visited map[string]bool) {
	// Prevent infinite recursion
	if visited[role.Name] {
		return
	}
	visited[role.Name] = true

	// Add direct permissions
	for _, perm := range role.Permissions {
		permissions[perm] = true
	}

	// Add inherited permissions
	for _, inheritedRoleName := range role.Inherits {
		if inheritedRole, exists := rm.roles[inheritedRoleName]; exists {
			rm.collectRolePermissions(inheritedRole, permissions, visited)
		}
	}
}

// AddRole adds a new role
func (rm *RBACManager) AddRole(role *Role) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.roles[role.Name]; exists {
		return fmt.Errorf("role already exists: %s", role.Name)
	}

	rm.roles[role.Name] = role
	return nil
}

// AddPermission adds a new permission
func (rm *RBACManager) AddPermission(permission *Permission) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.permissions[permission.Name]; exists {
		return fmt.Errorf("permission already exists: %s", permission.Name)
	}

	rm.permissions[permission.Name] = permission
	return nil
}

// RemoveRole removes a role
func (rm *RBACManager) RemoveRole(name string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.roles[name]; !exists {
		return fmt.Errorf("role not found: %s", name)
	}

	delete(rm.roles, name)
	return nil
}

// RemovePermission removes a permission
func (rm *RBACManager) RemovePermission(name string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.permissions[name]; !exists {
		return fmt.Errorf("permission not found: %s", name)
	}

	delete(rm.permissions, name)
	return nil
}

// ValidateRole validates a role configuration
func (rm *RBACManager) ValidateRole(role *Role) error {
	// Check if all permissions exist
	for _, permName := range role.Permissions {
		if _, exists := rm.permissions[permName]; !exists {
			return fmt.Errorf("permission not found: %s", permName)
		}
	}

	// Check if all inherited roles exist
	for _, inheritedRoleName := range role.Inherits {
		if _, exists := rm.roles[inheritedRoleName]; !exists {
			return fmt.Errorf("inherited role not found: %s", inheritedRoleName)
		}
	}

	return nil
}

// ExportConfig exports the current RBAC configuration
func (rm *RBACManager) ExportConfig() (*RBACConfig, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	config := &RBACConfig{
		Roles:       make([]Role, 0, len(rm.roles)),
		Permissions: make([]Permission, 0, len(rm.permissions)),
	}

	for _, role := range rm.roles {
		config.Roles = append(config.Roles, *role)
	}

	for _, permission := range rm.permissions {
		config.Permissions = append(config.Permissions, *permission)
	}

	return config, nil
}

// SaveConfig saves the current configuration to a file
func (rm *RBACManager) SaveConfig(filename string) error {
	config, err := rm.ExportConfig()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetRoleHierarchy returns the role hierarchy
func (rm *RBACManager) GetRoleHierarchy() map[string][]string {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	hierarchy := make(map[string][]string)
	for name, role := range rm.roles {
		hierarchy[name] = role.Inherits
	}

	return hierarchy
}

// ToJSON converts the RBAC configuration to JSON
func (rm *RBACManager) ToJSON() ([]byte, error) {
	config, err := rm.ExportConfig()
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(config, "", "  ")
} 