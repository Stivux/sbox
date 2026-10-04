package core_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Stivux/sbox/internal/core"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestBackupVaultCopiesEncryptedBytes(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "subs.sbox")
	dst := core.BackupPath(src)
	must(t, core.SaveSubscriptions(src, core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{URL: "https://example.com/sub#lab", Title: "lab"}},
	}, "test-password"))
	must(t, core.BackupVault(src, dst))

	want, err := os.ReadFile(src)
	must(t, err)
	got, err := os.ReadFile(dst)
	must(t, err)
	if !bytes.Equal(want, got) {
		t.Fatal("backup bytes differ from vault")
	}
	if !core.IsEncrypted(got) {
		t.Fatal("backup is not encrypted")
	}
	info, err := os.Stat(dst)
	must(t, err)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup perm %o, want 0600", info.Mode().Perm())
	}
}

func TestSaveSubscriptionsKeepsPreviousBackup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "subs.sbox")
	must(t, core.SaveSubscriptions(path, core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{URL: "https://example.com/a", Title: "a"}},
	}, "pw"))
	firstBytes, err := os.ReadFile(path)
	must(t, err)
	must(t, core.SaveSubscriptions(path, core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{URL: "https://example.com/b", Title: "b"}},
	}, "pw"))

	bak, err := os.ReadFile(core.BackupPath(path))
	must(t, err)
	if !bytes.Equal(bak, firstBytes) {
		t.Fatal("bak is not the previous vault generation")
	}
	loaded, err := core.LoadSubscriptions(core.BackupPath(path), "pw")
	must(t, err)
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
