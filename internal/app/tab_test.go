package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kooler/MiddayCommander/internal/config"
	"github.com/kooler/MiddayCommander/internal/remote/testserver"
	"github.com/kooler/MiddayCommander/internal/ui/panel"
	"github.com/kooler/MiddayCommander/internal/ui/tablist"
	"github.com/kooler/MiddayCommander/internal/vfs"
	"github.com/kooler/MiddayCommander/internal/vfs/local"
)

func openTabList(t *testing.T, m Model) Model {
	t.Helper()
	next, _ := m.startTabs()
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("startTabs returned %T", next)
	}
	return updated
}

func TestClosingATabReleasesItsConnections(t *testing.T) {
	isolate(t)
	srv := testserver.Start(t)
	m := newModel(t)
	m = connectPanel(t, m, testServerFor(t, srv))
	conn := m.tab().panelConns[m.tab().focus]
	if conn == nil {
		t.Fatal("want the connection recorded on the active tab")
	}

	// A second, empty tab so the first one can be closed.
	m.addTabFrom(m.activeTab)
	if _, err := conn.FS().ReadDir(srv.Root); err != nil {
		t.Fatal("the connection should still be open before closing its tab")
	}

	// The list stays open so more tabs can be closed in a row.
	m = openTabList(t, m)
	m, _ = run(t, m, tablist.CloseMsg{Index: 0})
	if _, err := conn.FS().ReadDir(srv.Root); err == nil {
		t.Error("closing the tab should release its connection")
	}
	if len(m.tabs) != 1 {
		t.Fatalf("want one tab left, got %d", len(m.tabs))
	}
	if m.tablist == nil {
		t.Error("the tab list should stay open after closing a tab")
	}
}

func TestClosingTabsInARow(t *testing.T) {
	isolate(t)
	m := newModel(t)
	m.addTabFrom(m.activeTab)
	m.addTabFrom(m.activeTab)
	if len(m.tabs) != 3 {
		t.Fatalf("want three tabs, got %d", len(m.tabs))
	}
	m = openTabList(t, m)

	for i := 0; i < 2; i++ {
		m, _ = run(t, m, tablist.CloseMsg{Index: 0})
		if m.tablist == nil {
			t.Fatalf("iteration %d: the tab list should stay open", i)
		}
	}
	if len(m.tabs) != 1 {
		t.Fatalf("want the single remaining tab, got %d", len(m.tabs))
	}

	// The last tab cannot be closed.
	m, _ = run(t, m, tablist.CloseMsg{Index: 0})
	if len(m.tabs) != 1 {
		t.Fatalf("the last tab cannot be closed, got %d", len(m.tabs))
	}
	if m.tablist == nil {
		t.Error("the tab list should stay open")
	}
}

func TestTabListNewAndJump(t *testing.T) {
	isolate(t)
	m := newModel(t)
	if len(m.tabs) != 1 {
		t.Fatalf("want one tab, got %d", len(m.tabs))
	}

	m = openTabList(t, m)
	m, _ = run(t, m, tablist.NewMsg{})
	if len(m.tabs) != 2 || m.activeTab != 1 {
		t.Fatalf("want two tabs with the new one active, got %d at %d", len(m.tabs), m.activeTab)
	}

	m = openTabList(t, m)
	m, _ = run(t, m, tablist.JumpMsg{Index: 0})
	if m.activeTab != 0 {
		t.Fatalf("want tab 0 after jump, got %d", m.activeTab)
	}
	if m.tablist != nil {
		t.Error("the dialog should close after a jump")
	}
}

func TestNewTabFromCursorTab(t *testing.T) {
	isolate(t)
	m := newModel(t)
	other := t.TempDir()
	m.addTabFrom(m.activeTab)
	if len(m.tabs) != 2 {
		t.Fatalf("want two tabs, got %d", len(m.tabs))
	}
	// Give tab 0 a distinct path, then request a new tab from its row.
	m.tabs[0].leftPanel.SetPath(other)
	m.openTabList(0)
	m, _ = run(t, m, tablist.NewMsg{From: 0})
	if len(m.tabs) != 3 {
		t.Fatalf("want three tabs, got %d", len(m.tabs))
	}
	if got := m.tabs[2].leftPanel.LocalPath(); got != other {
		t.Errorf("want the new tab based on tab 0's path %q, got %q", other, got)
	}
}

func TestAbbreviatePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot find the home directory: %v", err)
	}
	deep := home + "/Dev/go/MiddayCommander"

	// Home-relative abbreviations, within a 12-rune label.
	if got := abbreviatePath(home, 12); got != "~" {
		t.Errorf("home dir: got %q, want %q", got, "~")
	}
	if got := abbreviatePath(home+"/x", 12); got != "~/x" {
		t.Errorf("short home path: got %q, want %q", got, "~/x")
	}
	if got := abbreviatePath(deep, 12); got != "~/D/g/Midda…" {
		t.Errorf("deep home path: got %q", got)
	}

	// Non-home absolute paths keep their leading separator.
	if got := abbreviatePath("/usr/bin", 12); got != "/u/bin" {
		t.Errorf("short abs path: got %q, want %q", got, "/u/bin")
	}
	if got := abbreviatePath("/usr/local/lib/python3.12", 12); got != "/u/l/l/pyth…" {
		t.Errorf("long abs path: got %q, want %q", got, "/u/l/l/pyth…")
	}

	// Narrow labels truncate harder; a zero width yields nothing.
	if got := abbreviatePath(deep, 6); got != "~/D/g…" {
		t.Errorf("narrow label: got %q, want %q", got, "~/D/g…")
	}
	if got := abbreviatePath(deep, 0); got != "" {
		t.Errorf("zero width: got %q", got)
	}

	// Trivial inputs.
	if got := abbreviatePath("", 12); got != "untitled" {
		t.Errorf("empty: got %q, want %q", got, "untitled")
	}
	if got := abbreviatePath(".", 12); got != "." {
		t.Errorf("dot: got %q, want %q", got, ".")
	}
	if got := abbreviatePath("..", 12); got != ".." {
		t.Errorf("dotdot: got %q, want %q", got, "..")
	}
}

func TestTabBarVisibility(t *testing.T) {
	isolate(t)
	m := newModel(t)

	// A single tab shows no bar at all.
	if bar := m.renderTabBar(m.theme, m.width); bar != "" {
		t.Fatalf("want the bar hidden for a single tab, got %q", bar)
	}

	m.addTabFrom(m.activeTab)
	bar := m.renderTabBar(m.theme, m.width)
	if bar == "" {
		t.Fatal("want the bar shown for two tabs")
	}

	// The bar always fits: ten tabs at 100 columns give 10 cells each.
	for i := 0; i < 8; i++ {
		m.addTabFrom(m.activeTab)
	}
	bar = m.renderTabBar(m.theme, m.width)
	if lipgloss.Width(bar) != m.width {
		t.Errorf("want the bar to span exactly %d cells, got %d", m.width, lipgloss.Width(bar))
	}
}

func TestMaxTabsLimit(t *testing.T) {
	isolate(t)
	m := newModel(t)
	for i := 0; i < 20; i++ {
		m.addTabFrom(m.activeTab)
	}
	if len(m.tabs) != maxTabs {
		t.Fatalf("want the tab list capped at %d, got %d", maxTabs, len(m.tabs))
	}
}

func TestDirLoadedReachesItsTabWhenInactive(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shown.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(t)
	// Start a load on tab 0, then switch away before it lands.
	m.tabs[0].leftPanel.SetPath(dir)
	cmd := m.tabs[0].leftPanel.LoadDir()
	m.addTabFrom(m.activeTab)
	if m.activeTab == 0 {
		t.Fatal("want the new tab to be active")
	}

	// It lands in tab 0, not the active tab.
	m, _ = run(t, m, cmd())
	if got := m.tabs[0].leftPanel.Path(); got != dir {
		t.Fatalf("tab 0 left path = %q, want %q", got, dir)
	}
	if view := m.tabs[0].leftPanel.View(m.theme); !strings.Contains(view, "shown.txt") {
		t.Errorf("the load should have populated the inactive tab; view=%q", view)
	}
	if got := m.tabs[1].leftPanel.View(m.theme); strings.Contains(got, "shown.txt") {
		t.Error("the active tab must not receive another tab's load")
	}
}

// A load with no panel ID addresses every tab, not just the active one. The
// loop must keep going after the first panel accepts.
func TestDirLoadedWithoutAnIDReachesEveryTab(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(t)
	m.tabs[0].leftPanel.SetPath(dir)
	m.addTabFrom(m.activeTab)
	m.tabs[1].leftPanel.SetPath(dir)
	m.tabs[1].rightPanel.SetPath(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	m, _ = run(t, m, panel.DirLoadedMsg{Path: dir, Entries: entries})

	for _, tc := range []struct {
		tab  int
		side FocusTarget
	}{
		{0, FocusLeft},
		{1, FocusLeft},
		{1, FocusRight},
	} {
		p := m.tabs[tc.tab].panelFor(tc.side)
		if view := p.View(m.theme); !strings.Contains(view, "shared.txt") {
			t.Errorf("tab %d %v: the broadcast load should have reached this panel", tc.tab, tc.side)
		}
	}
}

func TestTabLabelShowsRemoteLocation(t *testing.T) {
	lfs := local.New(string(filepath.Separator))
	p := panel.New(lfs, "/", panel.KeyMap{}, config.Default())
	p.SetLocation(vfs.Location{FS: lfs, Path: "/var/log", Kind: vfs.KindSSH, Label: "ssh://kk@host/"})
	tabs := tab{leftPanel: p, rightPanel: p, focus: FocusLeft}
	if got := tabs.label(24); !strings.Contains(got, "ssh://kk@host") {
		t.Errorf("a remote tab label should show the ssh location, got %q", got)
	}
}

// panelFor is the only place a FocusTarget turns into a panel field, so a
// mismatch between side and field is worth pinning down directly.
func TestPanelForNamesTheRightField(t *testing.T) {
	lfs := local.New(string(filepath.Separator))
	left := panel.New(lfs, "/left", panel.KeyMap{}, config.Default())
	right := panel.New(lfs, "/right", panel.KeyMap{}, config.Default())
	tab := tab{leftPanel: left, rightPanel: right, focus: FocusLeft}

	for _, tc := range []struct {
		side FocusTarget
		want string
	}{
		{FocusLeft, "/left"},
		{FocusRight, "/right"},
	} {
		if got := tab.panelFor(tc.side).Path(); got != tc.want {
			t.Errorf("panelFor(%v) = %q, want %q", tc.side, got, tc.want)
		}
	}

	if got := FocusLeft.oppositeSide(); got != FocusRight {
		t.Errorf("FocusLeft.oppositeSide() = %v, want FocusRight", got)
	}
	if got := FocusRight.oppositeSide(); got != FocusLeft {
		t.Errorf("FocusRight.oppositeSide() = %v, want FocusLeft", got)
	}
}

func TestTabBarFitsNarrowWindow(t *testing.T) {
	isolate(t)
	m := newModel(t)
	for i := 0; i < 9; i++ {
		m.addTabFrom(m.activeTab)
	}
	for _, w := range []int{20, 30, 40, 59} {
		bar := m.renderTabBar(m.theme, w)
		if got := lipgloss.Width(bar); got != w {
			t.Errorf("width %d: bar spans %d cells, want %d", w, got, w)
		}
	}
}

func TestRestoreCursorReachesItsTabWhenInactive(t *testing.T) {
	isolate(t)
	dirA := t.TempDir()
	sub := filepath.Join(dirA, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModel(t)
	m.tabs[0].leftPanel.SetPath(dirA)
	m, _ = run(t, m, drain(t, m.tabs[0].leftPanel.LoadDir()))

	m.tabs[0].leftPanel.SetPath(sub)
	m, _ = run(t, m, drain(t, m.tabs[0].leftPanel.LoadDir()))

	// Go up, which queues a load then a cursor restore for this panel.
	backCmd := m.tabs[0].leftPanel.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	// Switch tabs before either message lands.
	m.addTabFrom(m.activeTab)
	if m.activeTab == 0 {
		t.Fatal("want the new tab to be active")
	}

	// tea.Sequence's message is an unexported []tea.Cmd; drive its two
	// steps by hand.
	var restore panel.RestoreCursorMsg
	seq := reflect.ValueOf(backCmd())
	for i := 0; i < seq.Len(); i++ {
		c, _ := seq.Index(i).Interface().(tea.Cmd)
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case panel.DirLoadedMsg:
			m, _ = run(t, m, msg)
		case panel.RestoreCursorMsg:
			restore = msg
		}
	}
	if restore.Name != "sub" {
		t.Fatalf("want RestoreCursorMsg{Name:sub}, got %+v", restore)
	}
	if m.tabs[0].leftPanel.Path() != dirA {
		t.Fatalf("tab 0 should have gone up to %q, got %q", dirA, m.tabs[0].leftPanel.Path())
	}

	// The restore lands in tab 0; the active tab's panels stay untouched.
	m, _ = run(t, m, restore)
	if e := m.tabs[0].leftPanel.CurrentEntry(); e == nil || e.Name() != "sub" {
		t.Errorf("tab 0 cursor should be on sub, got %v", e)
	}
	if m.tabs[1].leftPanel.Path() == sub {
		t.Error("the active tab must not receive another tab's go back")
	}
}

func TestQuickViewFollowsTabSwitch(t *testing.T) {
	isolate(t)
	dirA, dirB := t.TempDir(), t.TempDir()
	writeFile := func(dir, name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("hello "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(dirA, "alpha.txt")
	writeFile(dirB, "beta.txt")

	m := newModel(t)
	m.tabs[0].leftPanel.SetPath(dirA)
	m, _ = run(t, m, drain(t, m.tabs[0].leftPanel.LoadDir()))
	m = selectEntry(t, m, "alpha.txt")
	m.openQuickView()
	if m.tab().quickview == nil {
		t.Fatal("want a quick view on tab 0")
	}
	if got := m.tab().quickview.Path(); !strings.Contains(got, "alpha.txt") {
		t.Fatalf("preview should show alpha.txt, got %q", got)
	}

	// A second tab starts without a preview of its own.
	m.addTabFrom(m.activeTab)
	m.tabs[1].leftPanel.SetPath(dirB)
	m, _ = run(t, m, drain(t, m.tabs[1].leftPanel.LoadDir()))
	if m.tab().quickview != nil {
		t.Fatal("a fresh tab must not inherit a preview")
	}

	// Tab 0 keeps its preview while inactive.
	if m.tabs[0].quickview == nil || !strings.Contains(m.tabs[0].quickview.Path(), "alpha.txt") {
		t.Fatal("tab 0 preview must survive while another tab is active")
	}

	// Switching back brings tab 0's preview back as it was.
	m, _ = run(t, m, tablist.JumpMsg{Index: 0})
	if m.tab().quickview == nil || !strings.Contains(m.tab().quickview.Path(), "alpha.txt") {
		t.Fatalf("tab 0 preview should reappear on jump, got %+v", m.tab().quickview)
	}
	if m.tab().quickFocus {
		t.Error("quickFocus should be restored with the tab")
	}

	// Closing the preview closes it only in the active tab.
	m.closeQuickView()
	if m.tabs[0].quickview != nil {
		t.Error("close should drop the active tab's preview")
	}
}

func TestClosingTabWithPreviewKeepsOthers(t *testing.T) {
	isolate(t)
	m := newModel(t)
	m.openQuickView()
	if m.tab().quickview == nil {
		t.Fatal("want a quick view on tab 0")
	}
	m.addTabFrom(m.activeTab)
	m.closeTabAt(1)
	if len(m.tabs) != 1 {
		t.Fatalf("want one tab left, got %d", len(m.tabs))
	}
	if m.tab().quickview == nil {
		t.Error("closing another tab must not close this tab's preview")
	}
}

func TestSharedConnectionSurvivesClosingOneOfTwoTabs(t *testing.T) {
	isolate(t)
	srv := testserver.Start(t)
	server := testServerFor(t, srv)

	// The same host in both tabs shares one refcounted connection.
	m := newModel(t)
	m = connectPanel(t, m, server)
	conn := m.tab().panelConns[m.tab().focus]
	m.addTabFrom(m.activeTab)
	m = connectPanel(t, m, server)
	if got := m.tab().panelConns[m.tab().focus]; got != conn {
		t.Fatal("want both tabs to share one connection")
	}

	// Closing one tab leaves the connection to the other one.
	m = openTabList(t, m)
	m, _ = run(t, m, tablist.CloseMsg{Index: 1})
	if len(m.tabs) != 1 {
		t.Fatalf("want one tab, got %d", len(m.tabs))
	}
	if _, err := conn.FS().ReadDir(srv.Root); err != nil {
		t.Error("closing one tab must not close the connection the other tab uses")
	}
	if m.tab().panelConns[m.tab().focus] != conn {
		t.Error("the surviving tab should still track the connection")
	}
}

func TestConnectionReleasedWhenBothPanelsLeaveTheServer(t *testing.T) {
	isolate(t)
	srv := testserver.Start(t)
	server := testServerFor(t, srv)

	m := newModel(t)
	m = connectPanel(t, m, server)
	conn := m.tab().panelConns[FocusLeft]

	// Opening the same host in the other panel hands back the same conn.
	m, _ = run(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m = connectPanel(t, m, server)
	if got := m.tab().panelConns[FocusRight]; got != conn {
		t.Fatal("want the shared connection in the right panel too")
	}

	// Leaving the server in one panel drops one reference, not the conn.
	m.tab().rightPanel.SetPath(t.TempDir())
	m.releaseUnusedConnections()
	if _, err := conn.FS().ReadDir(srv.Root); err != nil {
		t.Error("the conn should stay for the left panel")
	}

	// Leaving in the second panel closes it.
	m.tab().leftPanel.SetPath(t.TempDir())
	m.releaseUnusedConnections()
	if _, err := conn.FS().ReadDir(srv.Root); err == nil {
		t.Error("the conn should close once no panel uses it")
	}
}
