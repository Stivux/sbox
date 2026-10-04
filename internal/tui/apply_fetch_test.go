package tui //nolint:testpackage // applyFetch is unexported

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rivo/tview"

	"github.com/Stivux/sbox/internal/core"
)

func TestApplyFetchAllFailedDoesNotSave(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "subscriptions.sbox")
	cfg := core.SubscriptionsConfig{Subscriptions: []core.Subscription{{
		URL: "https://example.com/sub", Title: "lab",
	}}}
	if err := core.SaveSubscriptions(path, cfg, "pw"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	a := &App{path: path, password: "pw", config: cfg, status: tview.NewTextView()}
	a.applyFetch([]fetchResult{{err: errors.New("boom")}}, true)

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("vault rewritten after a failed fetch")
	}
	if _, err = os.Stat(core.BackupPath(path)); !os.IsNotExist(err) {
		t.Fatalf("backup created: %v", err)
	}
}
