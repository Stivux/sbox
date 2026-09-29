package core_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Stivux/sbox/internal/core"
)

func TestBackupVaultCopiesEncryptedBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "subs.sbox")
	dst := core.BackupPath(src)

	cfg := core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{
			URL:   "https://example.com/sub#lab",
			Title: "lab",
		}},
	}
	if err := core.SaveSubscriptions(src, cfg, "test-password"); err != nil {
		t.Fatalf("SaveSubscriptions: %v", err)
	}

	if err := core.BackupVault(src, dst); err != nil {
		t.Fatalf("BackupVault: %v", err)
	}

	want, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Fatal("backup bytes differ from vault")
	}
	if !core.IsEncrypted(got) {
		t.Fatal("backup is not encrypted")
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup perm %o, want 0600", info.Mode().Perm())
	}
}

func TestSaveSubscriptionsKeepsPreviousBackup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "subs.sbox")
	first := core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{URL: "https://example.com/a", Title: "a"}},
	}
	second := core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{URL: "https://example.com/b", Title: "b"}},
	}

	if err := core.SaveSubscriptions(path, first, "pw"); err != nil {
		t.Fatalf("first save: %v", err)
	}
	firstBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err = core.SaveSubscriptions(path, second, "pw"); err != nil {
		t.Fatalf("second save: %v", err)
	}

	bak, err := os.ReadFile(core.BackupPath(path))
	if err != nil {
		t.Fatalf("read bak: %v", err)
	}
	if !bytes.Equal(bak, firstBytes) {
		t.Fatal("bak is not the previous vault generation")
	}

	loaded, err := core.LoadSubscriptions(core.BackupPath(path), "pw")
	if err != nil {
		t.Fatalf("load bak: %v", err)
	}
	if loaded.Subscriptions[0].Title != "a" {
		t.Fatalf("bak title %q, want a", loaded.Subscriptions[0].Title)
	}
}

func TestBackupVaultSamePath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "subs.sbox")
	if err := core.BackupVault(path, path); err == nil {
		t.Fatal("expected error for identical paths")
	}
}
