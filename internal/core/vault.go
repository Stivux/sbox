package core

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	vaultFilePerm     = 0o600
	outboundsFilePerm = 0o644
	backupSuffix      = ".bak"
)

// BackupPath is the sidecar replica of the encrypted vault.
func BackupPath(path string) string { return path + backupSuffix }

// BackupVault copies the encrypted vault from src to dest without decrypting it.
func BackupVault(src, dest string) error {
	if src == dest {
		return fmt.Errorf("backup vault: source and destination are the same")
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("backup vault: read: %w", err)
	}
	// #nosec G703 -- dest is a caller-provided path, not user-controlled input
	if err = os.WriteFile(dest, data, vaultFilePerm); err != nil {
		return fmt.Errorf("backup vault: write: %w", err)
	}
	return nil
}

// SaveSubscriptions encrypts cfg and writes it to path. An existing file is copied to BackupPath first.
func SaveSubscriptions(path string, cfg SubscriptionsConfig, password string) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("save subscription: marshal: %w", err)
	}
	enc, err := Encrypt(data, password)
	if err != nil {
		return fmt.Errorf("save subscription: encrypt: %w", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		if err = BackupVault(path, BackupPath(path)); err != nil {
			return fmt.Errorf("save subscription: backup: %w", err)
		}
	}
	if err = os.WriteFile(path, enc, vaultFilePerm); err != nil {
		return fmt.Errorf("save subscription: write: %w", err)
	}
	return nil
}

// LoadSubscriptions decrypts the vault at path. A missing file yields an empty config.
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
