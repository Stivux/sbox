// Package tui implements the interactive terminal UI for managing VPN subscriptions.
package tui

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Stivux/sbox/internal/core"
)

var errNoServers = errors.New("no servers")

const (
	// DefaultSubscriptionsPath is the default encrypted subscriptions file name.
	DefaultSubscriptionsPath = "subscriptions.sbox"
	// DefaultOutboundsPath is written when a server is selected with Enter.
	DefaultOutboundsPath = "outbounds.json"

	tabAll     = "all"
	subTabPref = "sub-"

	rowH   = 1
	formW  = 48
	fieldW = 32
	labelH = 2
	errH   = 2
)

const (
	modePassword = iota
	modeMain
	modeAddURL
)

// serverRef points at an outbound inside App.config.
type serverRef struct{ sub, out int }

// App is the root TUI application.
type App struct {
	ui       *tview.Application
	mode     int
	busy     atomic.Bool
	password string
	path     string
	outPath  string
	config   core.SubscriptionsConfig

	root   *tview.Flex
	pages  *tview.Pages
	tabs   *tview.TextView
	status *tview.TextView
	urlIn  *tview.InputField

	tabIDs   []string
	active   string
	pageRefs map[string][]serverRef
}

// Run starts the TUI. Empty path uses DefaultSubscriptionsPath.
func Run(path string) error {
	if path == "" {
		path = DefaultSubscriptionsPath
	}
	a := &App{
		ui:       tview.NewApplication(),
		path:     path,
		outPath:  DefaultOutboundsPath,
		mode:     modePassword,
		pageRefs: make(map[string][]serverRef),
	}
	a.ui.SetInputCapture(a.onKey)
	a.showPassword()
	return a.ui.EnableMouse(true).EnablePaste(true).Run()
}

func (a *App) onKey(ev *tcell.EventKey) *tcell.EventKey {
	if a.mode != modeMain {
		return ev
	}
	switch {
	case ev.Key() == tcell.KeyLeft:
		a.switchTab(-1)
		return nil
	case ev.Key() == tcell.KeyRight:
		a.switchTab(1)
		return nil
	case ev.Key() == tcell.KeyRune:
		if a.busy.Load() {
			return ev
		}
		switch ev.Rune() {
		case 'a', 'A':
			a.showAddURL()
		case 'd', 'D':
			a.deleteSub()
		case 'u', 'U':
			a.refresh()
		case 'q', 'Q':
			a.ui.Stop()
		default:
			return ev
		}
		return nil
	default:
		return ev
	}
}

func (a *App) showPassword() {
	a.mode = modePassword
	isNew := !fileExists(a.path)

	label := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true)
	errView := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true)
	pass := maskedField("Password: ")
	confirm := maskedField("Confirm:  ")

	form := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(label, labelH, 0, false).
		AddItem(pass, rowH, 0, true)

	if isNew {
		label.SetText("Create a password for your subscription vault\n(first run — no vault file found)")
		form.AddItem(confirm, rowH, 0, false)
	} else {
		label.SetText("Enter password to unlock subscriptions")
	}
	form.AddItem(errView, errH, 0, false).AddItem(nil, 0, 1, false)

	pass.SetDoneFunc(func(key tcell.Key) {
		a.onPasswordDone(key, isNew, pass, confirm, errView)
	})
	confirm.SetDoneFunc(func(key tcell.Key) {
		a.onConfirmDone(key, pass, confirm, errView)
	})

	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(form, formW, 0, true).
		AddItem(nil, 0, 1, false)
	a.ui.SetRoot(centered, true).SetFocus(pass)
}

func (a *App) onPasswordDone(
	key tcell.Key, isNew bool,
	pass, confirm *tview.InputField, errView *tview.TextView,
) {
	if key == tcell.KeyEscape {
		a.ui.Stop()
		return
	}
	if key == tcell.KeyTab && isNew {
		a.ui.SetFocus(confirm)
		return
	}
	if key != tcell.KeyEnter {
		return
	}
	if isNew {
		if pass.GetText() == "" {
			errView.SetText("[red]Password is required[-]")
			return
		}
		a.ui.SetFocus(confirm)
		return
	}
	a.unlock(pass.GetText(), pass, errView)
}

func (a *App) onConfirmDone(key tcell.Key, pass, confirm *tview.InputField, errView *tview.TextView) {
	if key == tcell.KeyEscape {
		a.ui.Stop()
		return
	}
	if key == tcell.KeyBacktab {
		a.ui.SetFocus(pass)
		return
	}
	if key != tcell.KeyEnter {
		return
	}
	p, c := pass.GetText(), confirm.GetText()
	if p == "" {
		errView.SetText("[red]Password is required[-]")
		a.ui.SetFocus(pass)
		return
	}
	if p != c {
		errView.SetText("[red]Passwords do not match[-]")
		confirm.SetText("")
		return
	}
	a.password, a.config = p, core.SubscriptionsConfig{}
	if err := core.SaveSubscriptions(a.path, a.config, a.password); err != nil {
		errView.SetText(fmt.Sprintf("[red]Failed to create vault: %v[-]", err))
		return
	}
	a.showMain()
}

func (a *App) unlock(password string, input *tview.InputField, errView *tview.TextView) {
	if password == "" {
		errView.SetText("[red]Password is required[-]")
		return
	}
	cfg, err := core.LoadSubscriptions(a.path, password)
	if err != nil {
		errView.SetText(fmt.Sprintf("[red]%v[-]", err))
		input.SetText("")
		return
	}
	a.password, a.config = password, cfg
	a.showMain()
}

func (a *App) showMain() {
	a.mode = modeMain
	a.pages = tview.NewPages()
	a.tabs = tview.NewTextView().
		SetDynamicColors(true).SetRegions(true).SetWrap(false).
		SetHighlightedFunc(func(added, _, _ []string) {
			if len(added) == 0 {
				return
			}
			a.active = added[0]
			a.pages.SwitchToPage(added[0])
			a.focusList()
		})
	a.status = tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	a.urlIn = tview.NewInputField().SetLabel("Subscription URL: ").SetFieldWidth(0)
	a.urlIn.SetDoneFunc(a.onURLDone)

	a.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.tabs, rowH, 0, false).
		AddItem(a.status, rowH, 0, false)

	a.ui.SetRoot(a.root, true)
	a.rebuild()
	a.focusList()
}

func (a *App) rebuild() {
	for _, name := range a.pages.GetPageNames(true) {
		a.pages.RemovePage(name)
	}
	a.tabs.Clear()
	a.tabIDs = a.tabIDs[:0]
	clear(a.pageRefs)

	all := newList()
	a.addTab(tabAll, tabAll, all, true)

	for i, sub := range a.config.Subscriptions {
		id := subTabPref + strconv.Itoa(i)
		list := newList()
		title := sub.DisplayTitle()
		refs := make([]serverRef, 0, len(sub.Outbounds))

		for j, entry := range sub.Outbounds {
			ref := serverRef{sub: i, out: j}
			refs = append(refs, ref)
			name := entry.DisplayName()
			list.AddItem(name, "", 0, a.selectFn(ref))
			all.AddItem(fmt.Sprintf("[%s] %s", title, name), "", 0, a.selectFn(ref))
			a.pageRefs[tabAll] = append(a.pageRefs[tabAll], ref)
		}
		a.pageRefs[id] = refs
		a.addTab(id, title, list, false)
	}

	if a.active == "" || !slices.Contains(a.tabIDs, a.active) {
		a.active = tabAll
	}
	a.tabs.Highlight(a.active).ScrollToHighlight()
	a.pages.SwitchToPage(a.active)
	a.statusMsgf("%s", a.help())
}

func (a *App) addTab(id, title string, page tview.Primitive, show bool) {
	a.pages.AddPage(id, page, true, show)
	a.tabIDs = append(a.tabIDs, id)
	_, _ = fmt.Fprintf(a.tabs, `["%s"][darkcyan]%s[white][""]  `, id, title)
}

func (a *App) selectFn(ref serverRef) func() {
	return func() { a.writeOutbound(ref) }
}

func (a *App) switchTab(delta int) {
	if len(a.tabIDs) == 0 {
		return
	}
	i := max(slices.Index(a.tabIDs, a.active), 0)
	i = (i + delta + len(a.tabIDs)) % len(a.tabIDs)
	a.tabs.Highlight(a.tabIDs[i]).ScrollToHighlight()
}

func (a *App) focusList() {
	if name, p := a.pages.GetFrontPage(); name != "" && p != nil {
		a.ui.SetFocus(p)
	}
}

func (a *App) showAddURL() {
	a.mode = modeAddURL
	a.root.RemoveItem(a.status)
	a.root.AddItem(a.urlIn, rowH, 0, true)
	a.urlIn.SetText("")
	a.ui.SetFocus(a.urlIn)
}

func (a *App) hideAddURL() {
	a.mode = modeMain
	a.root.RemoveItem(a.urlIn)
	a.root.AddItem(a.status, rowH, 0, false)
	a.focusList()
}

func (a *App) onURLDone(key tcell.Key) {
	if key == tcell.KeyEscape {
		a.hideAddURL()
		return
	}
	if key != tcell.KeyEnter {
		return
	}
	u := a.urlIn.GetText()
	a.hideAddURL()
	if u != "" {
		a.runFetch([]string{u}, "Fetching subscription", true)
	}
}

func (a *App) writeOutbound(ref serverRef) {
	if a.busy.Load() {
		return
	}
	entry, ok := a.entryAt(ref)
	if !ok {
		a.statusMsgf("[red]Invalid server selection[-]")
		return
	}
	if err := core.WriteOutboundsConfig(a.outPath, entry.Outbound); err != nil {
		a.statusMsgf("[red]Write %s failed: %v[-]", a.outPath, err)
		return
	}
	a.statusMsgf("[green]Wrote %s ← %s[-]  |  %s", a.outPath, entry.DisplayName(), a.help())
}

func (a *App) deleteSub() {
	idx, ok := a.targetSub("delete")
	if !ok {
		return
	}
	title := a.config.Subscriptions[idx].DisplayTitle()
	a.config.RemoveAt(idx)
	if a.active != tabAll {
		a.active = tabAll
	}
	if err := a.save(); err != nil {
		a.rebuild()
		a.focusList()
		a.statusMsgf("[red]Delete failed to save: %v[-]", err)
		return
	}
	a.rebuild()
	a.focusList()
	a.statusMsgf("[green]Deleted subscription %q[-]  |  %s", title, a.help())
}

func (a *App) refresh() {
	if a.busy.Load() {
		return
	}
	if a.active == tabAll {
		urls := make([]string, 0, len(a.config.Subscriptions))
		for _, s := range a.config.Subscriptions {
			if s.URL != "" {
				urls = append(urls, s.URL)
			}
		}
		if len(urls) == 0 {
			a.statusMsgf("[red]No subscription URLs to refresh[-]")
			return
		}
		a.runFetch(urls, fmt.Sprintf("Refreshing %d subscription(s)", len(urls)), false)
		return
	}

	idx, ok := parseSubIndex(a.active)
	if !ok || idx < 0 || idx >= len(a.config.Subscriptions) {
		a.statusMsgf("[red]Nothing to refresh[-]")
		return
	}
	sub := a.config.Subscriptions[idx]
	if sub.URL == "" {
		a.statusMsgf("[red]Subscription %q has no URL[-]", sub.DisplayTitle())
		return
	}
	a.runFetch([]string{sub.URL}, fmt.Sprintf("Refreshing %q", sub.DisplayTitle()), false)
}

type fetchResult struct {
	sub core.Subscription
	err error
}

func (a *App) runFetch(urls []string, pending string, addMode bool) {
	if !a.busy.CompareAndSwap(false, true) {
		return
	}
	a.statusMsgf("[yellow]%s…[-]", pending)

	go func() {
		out := make([]fetchResult, len(urls))
		for i, u := range urls {
			cfg, err := core.FetchSubscription(u)
			if err != nil {
				out[i].err = err
				continue
			}
			if len(cfg.Subscriptions) == 0 {
				out[i].err = errNoServers
				continue
			}
			out[i].sub = cfg.Subscriptions[0]
		}
		a.ui.QueueUpdateDraw(func() {
			a.busy.Store(false)
			a.applyFetch(out, addMode)
		})
	}()
}

func (a *App) applyFetch(results []fetchResult, addMode bool) {
	ok, fail := 0, 0
	var firstErr error
	var last core.Subscription
	for _, r := range results {
		if r.err != nil {
			fail++
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		a.config.Upsert(r.sub)
		last = r.sub
		ok++
	}

	if err := a.save(); err != nil && ok > 0 {
		a.rebuild()
		a.focusList()
		a.statusMsgf("[red]Updated in memory but save failed: %v[-]", err)
		return
	}
	a.rebuild()
	a.focusList()

	switch {
	case ok == 0:
		a.statusMsgf("[red]Failed: %v[-]", firstErr)
	case addMode && ok == 1:
		a.statusMsgf("[green]Added %q (%d servers)[-]  |  %s",
			last.DisplayTitle(), len(last.Outbounds), a.help())
	case fail == 0:
		a.statusMsgf("[green]Refreshed %d subscription(s)[-]  |  %s", ok, a.help())
	default:
		a.statusMsgf("[yellow]OK %d, failed %d[-]  |  %s", ok, fail, a.help())
	}
}

func (a *App) targetSub(verb string) (int, bool) {
	if idx, ok := parseSubIndex(a.active); ok {
		if idx < 0 || idx >= len(a.config.Subscriptions) {
			a.statusMsgf("[red]No subscription on this tab[-]")
			return 0, false
		}
		return idx, true
	}
	if a.active != tabAll {
		a.statusMsgf("[red]No subscription selected[-]")
		return 0, false
	}
	if len(a.config.Subscriptions) == 0 {
		a.statusMsgf("[red]No subscriptions to %s[-]", verb)
		return 0, false
	}
	_, prim := a.pages.GetFrontPage()
	list, _ := prim.(*tview.List)
	if list == nil || list.GetItemCount() == 0 {
		a.statusMsgf("[red]Select a server whose subscription to %s[-]", verb)
		return 0, false
	}
	item := list.GetCurrentItem()
	refs := a.pageRefs[tabAll]
	if item < 0 || item >= len(refs) {
		a.statusMsgf("[red]Select a server whose subscription to %s[-]", verb)
		return 0, false
	}
	return refs[item].sub, true
}

func (a *App) entryAt(ref serverRef) (core.OutboundEntry, bool) {
	if ref.sub < 0 || ref.sub >= len(a.config.Subscriptions) {
		return core.OutboundEntry{}, false
	}
	outs := a.config.Subscriptions[ref.sub].Outbounds
	if ref.out < 0 || ref.out >= len(outs) {
		return core.OutboundEntry{}, false
	}
	return outs[ref.out], true
}

func (a *App) save() error {
	return core.SaveSubscriptions(a.path, a.config, a.password)
}

func (a *App) help() string {
	return fmt.Sprintf(
		"[yellow]a[-] add  [yellow]d[-] delete  [yellow]u[-] refresh  [yellow]Enter[-] write  "+
			"[yellow]←/→[-] tabs  [yellow]q[-] quit  |  %d sub(s), %d server(s)",
		len(a.config.Subscriptions), a.config.ServerCount(),
	)
}

func (a *App) statusMsgf(format string, args ...any) {
	a.status.SetText(" " + fmt.Sprintf(format, args...))
}

func maskedField(label string) *tview.InputField {
	return tview.NewInputField().
		SetLabel(label).
		SetMaskCharacter('*').
		SetFieldWidth(fieldW)
}

func newList() *tview.List {
	// Disable style tags so names like "[sub] server" are literal text.
	// With tags on, "[title]" is parsed as a color and selected rows become
	// white-on-gray (unreadable) on the all tab.
	return tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedFocusOnly(false).
		SetWrapAround(true).
		SetUseStyleTags(false, false)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parseSubIndex(pageID string) (int, bool) {
	rest, ok := strings.CutPrefix(pageID, subTabPref)
	if !ok {
		return 0, false
	}
	idx, err := strconv.Atoi(rest)
	return idx, err == nil
}
