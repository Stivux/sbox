package tui //nolint:testpackage // unexported password-screen focus needs the simulation screen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestNewVaultCyclesFields(t *testing.T) {
	t.Parallel()
	ui, screen, stop := startPassword(t, filepath.Join(t.TempDir(), "subscriptions.sbox"))
	defer stop()

	waitStars(t, ui, screen, "Password:", 0)
	screen.InjectKey(tcell.KeyRune, 'a', tcell.ModNone)
	waitStars(t, ui, screen, "Password:", 1)

	screen.InjectKey(tcell.KeyDown, 0, tcell.ModNone)
	screen.InjectKey(tcell.KeyRune, 'b', tcell.ModNone)
	waitStars(t, ui, screen, "Confirm:", 1)
	if got := stars(snapshot(ui, screen), "Password:"); got != 1 {
		t.Fatalf("password stars = %d, want 1", got)
	}

	screen.InjectKey(tcell.KeyUp, 0, tcell.ModNone)
	screen.InjectKey(tcell.KeyRune, 'c', tcell.ModNone)
	waitStars(t, ui, screen, "Password:", 2)

	screen.InjectKey(tcell.KeyTab, 0, tcell.ModNone)
	screen.InjectKey(tcell.KeyRune, 'd', tcell.ModNone)
	waitStars(t, ui, screen, "Confirm:", 2)

	screen.InjectKey(tcell.KeyTab, 0, tcell.ModNone)
	screen.InjectKey(tcell.KeyRune, 'e', tcell.ModNone)
	waitStars(t, ui, screen, "Password:", 3)

	screen.InjectKey(tcell.KeyBacktab, 0, tcell.ModNone)
	screen.InjectKey(tcell.KeyRune, 'f', tcell.ModNone)
	waitStars(t, ui, screen, "Confirm:", 3)
	if got := stars(snapshot(ui, screen), "Password:"); got != 3 {
		t.Fatalf("password stars = %d, want 3", got)
	}
}

func TestUnlockScreenHasOneField(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "subscriptions.sbox")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}

	ui, screen, stop := startPassword(t, path)
	defer stop()
	waitText(t, ui, screen, "Enter password to unlock")

	screen.InjectKey(tcell.KeyRune, 'a', tcell.ModNone)
	waitStars(t, ui, screen, "Password:", 1)
	screen.InjectKey(tcell.KeyDown, 0, tcell.ModNone)
	screen.InjectKey(tcell.KeyTab, 0, tcell.ModNone)
	screen.InjectKey(tcell.KeyRune, 'b', tcell.ModNone)
	waitStars(t, ui, screen, "Password:", 2)
	if strings.Contains(snapshot(ui, screen), "Confirm:") {
		t.Fatal("unlock screen shows a confirm field")
	}
}

func startPassword(t *testing.T, path string) (*tview.Application, tcell.SimulationScreen, func()) {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	a := &App{
		ui:      tview.NewApplication().SetScreen(screen),
		path:    path,
		outPath: DefaultOutboundsPath,
		mode:    modePassword,
	}
	screen.SetSize(80, 24)
	a.ui.SetInputCapture(a.onKey)
	a.showPassword()

	done := make(chan struct{})
	go func() {
		_ = a.ui.EnableMouse(true).EnablePaste(true).Run()
		close(done)
	}()
	return a.ui, screen, func() {
		a.ui.Stop()
		<-done
	}
}

func waitStars(t *testing.T, ui *tview.Application, screen tcell.SimulationScreen, label string, want int) {
	t.Helper()
	waitUntil(t, ui, screen, func(text string) bool { return stars(text, label) == want })
}

func waitText(t *testing.T, ui *tview.Application, screen tcell.SimulationScreen, text string) {
	t.Helper()
	waitUntil(t, ui, screen, func(got string) bool { return strings.Contains(got, text) })
}

func waitUntil(t *testing.T, ui *tview.Application, screen tcell.SimulationScreen, ok func(string) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		last = snapshot(ui, screen)
		if ok(last) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out\n%s", last)
}

// snapshot reads the screen on the UI goroutine so it does not race with drawing.
func snapshot(ui *tview.Application, screen tcell.SimulationScreen) string {
	var text string
	ui.QueueUpdate(func() { text = screenText(screen) })
	return text
}

func stars(text, label string) int {
	for line := range strings.SplitSeq(text, "\n") {
		if strings.Contains(line, label) {
			return strings.Count(line, "*")
		}
	}
	return -1
}

func screenText(screen tcell.SimulationScreen) string {
	cells, w, h := screen.GetContents()
	var b strings.Builder
	for row := range h {
		line := make([]rune, w)
		for col := range w {
			r := cells[row*w+col].Runes
			if len(r) == 0 {
				line[col] = ' '
				continue
			}
			line[col] = r[0]
		}
		b.WriteString(strings.TrimRight(string(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}
