package tui //nolint:testpackage // onKey is unexported

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestQuitWhileBusy(t *testing.T) {
	t.Parallel()
	a := &App{ui: tview.NewApplication(), mode: modeMain}
	a.busy.Store(true)

	if got := a.onKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); got != nil {
		t.Fatal("q should quit while busy")
	}
	if got := a.onKey(tcell.NewEventKey(tcell.KeyRune, 'u', tcell.ModNone)); got == nil {
		t.Fatal("u should be ignored while busy")
	}
}
