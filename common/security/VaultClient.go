// VaultClient.go
package security

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// currentSecrets atomically stores the current secret bundle.
var currentSecrets atomic.Value

// VaultClient integrates with HashiCorp Vault for secure secret management.
type VaultClient struct {
	client *api.Client
	logger *zap.Logger
}

// NewVaultClient initializes a new VaultClient.
func NewVaultClient(address string, logger *zap.Logger) (*VaultClient, error) {
	config := api.DefaultConfig()
	config.Address = address
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &VaultClient{client: client, logger: logger}, nil
}

// RotateSecrets fetches and updates the current secret bundle.
func (v *VaultClient) RotateSecrets(ctx context.Context) error {
	secret, err := v.client.Logical().Read("secret/data/iaros")
	if err != nil {
		v.logger.Error("Failed to fetch secrets", zap.Error(err))
		return err
	}
	currentSecrets.Store(secret)
	v.logger.Info("Secrets rotated successfully")
	return nil
}
