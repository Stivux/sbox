package core

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	vaultFilePerm     = 0o600
	outboundsFilePerm = 0o644
)

// SaveSubscriptions encrypts cfg and writes it to path.
func SaveSubscriptions(path string, cfg SubscriptionsConfig, password string) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("save subscription: marshal: %w", err)
	}
	enc, err := Encrypt(data, password)
	if err != nil {
		return fmt.Errorf("save subscription: encrypt: %w", err)
	}
	if err = os.WriteFile(path, enc, vaultFilePerm); err != nil {
		return fmt.Errorf("save subscription: write: %w", err)
	}
	return nil
}

// LoadSubscriptions decrypts the vault at path. Missing file yields an empty config.
func LoadSubscriptions(path string, password string) (SubscriptionsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SubscriptionsConfig{}, nil
		}
		return SubscriptionsConfig{}, fmt.Errorf("load subscription: read: %w", err)
	}

	plain, err := Decrypt(data, password)
	if err != nil {
		return SubscriptionsConfig{}, fmt.Errorf("load subscription: decrypt: %w", err)
	}

	var result SubscriptionsConfig
	if err = json.Unmarshal(plain, &result); err != nil {
		return SubscriptionsConfig{}, fmt.Errorf("load subscription: unmarshal: %w", err)
	}
	return result, nil
}
