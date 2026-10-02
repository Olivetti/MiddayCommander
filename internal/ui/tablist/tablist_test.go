package tablist

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavigation(t *testing.T) {
	rows := []Row{
		{Number: 1, Left: "/one", Right: "/two", Active: true},
		{Number: 2, Left: "/three", Right: "/four"},
		{Number: 3, Left: "/five", Right: "/six"},
	}
	m := New(rows, 0, 100, 30, 3, 10)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if jump, ok := cmd().(JumpMsg); !ok || jump.Index != 2 {
		t.Fatalf("want JumpMsg{Index:2}, got %v", cmd())
	}

	m = New(rows, 1, 100, 30, 3, 10)
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if close, ok := cmd().(CloseMsg); !ok || close.Index != 1 {
		t.Fatalf("want CloseMsg{Index:1}, got %v", cmd())
	}

	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if newMsg, ok := cmd().(NewMsg); !ok || newMsg.From != 1 {
		t.Fatalf("want NewMsg{From:1}, got %v", cmd())
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := cmd().(DismissMsg); !ok {
		t.Fatalf("want DismissMsg, got %v", cmd())
	}
}

func TestFilter(t *testing.T) {
	rows := []Row{
		{Number: 1, Left: "/home/dev/go", Right: "/var/log"},
		{Number: 2, Left: "/srv/data", Right: "/etc"},
	}
	typeKey := func(m Model, r rune) Model {
		n, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		return n
	}

	m := New(rows, 0, 100, 30, 2, 10)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	if !m.filtering {
		t.Fatal("want filtering to start on f")
	}
	m = typeKey(m, 's')
	m = typeKey(m, 'r')
	if m.filter != "sr" || len(m.visible) != 1 || m.visible[0] != 1 {
		t.Fatalf("filter 'sr' should narrow to tab 2, got filter=%q visible=%v", m.filter, m.visible)
	}

	// j/k are typed into the query, not used for navigation.
	m = New(rows, 0, 100, 30, 2, 10)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = typeKey(m, 'j')
	m = typeKey(m, 'k')
	if m.filter != "jk" {
		t.Fatalf("want 'jk' typed into the query, got %q", m.filter)
	}
	if len(m.visible) != 0 {
		t.Fatalf("'jk' matches no path, want an empty list, got visible=%v", m.visible)
	}

	m = New(rows, 0, 100, 30, 2, 10)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = typeKey(m, 'a') // matches /var and /srv/data
	if m.cursor != 0 {
		t.Fatalf("want cursor at 0, got %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("want down arrow to move the cursor to 1, got %d", m.cursor)
	}

	m = New(rows, 0, 100, 30, 2, 10)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = typeKey(m, 's')
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.filtering || len(m.visible) != 2 {
		t.Fatalf("esc should clear the filter, got filtering=%v visible=%v", m.filtering, m.visible)
	}
}

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		name   string
		target string
		query  string
		want   bool
	}{
		{"subsequence", "/home/dev/go", "dev", true},
		{"wrong order", "/home/dev/go", "god", false},
		{"empty query", "anything", "", true},
		{"case insensitive", "/Home/Dev", "home", true},
	}
	for _, c := range cases {
		if got := fuzzyMatch(c.target, c.query); got != c.want {
			t.Errorf("%s: fuzzyMatch(%q,%q) = %v, want %v", c.name, c.target, c.query, got, c.want)
		}
	}
}

func TestDigitJump(t *testing.T) {
	rows := []Row{
		{Number: 1, Left: "/a", Right: "/b"},
		{Number: 2, Left: "/c", Right: "/d"},
		{Number: 3, Left: "/e", Right: "/f"},
	}
	m := New(rows, 0, 100, 30, 3, 10)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if jump, ok := cmd().(JumpMsg); !ok || jump.Index != 2 {
		t.Fatalf("want JumpMsg{Index:2} for '3', got %v", cmd())
	}

	m = New(rows, 0, 100, 30, 3, 10)
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")}); cmd != nil {
		t.Fatalf("want no jump for out-of-range digit, got %v", cmd())
	}
}

func TestFooterHints(t *testing.T) {
	// Two of ten tabs: both hints shown.
	rows := []Row{{Number: 1, Active: true}, {Number: 2}}
	out := New(rows, 0, 120, 40, 2, 10).View(120, 40)
	if !strings.Contains(out, "n:New") || !strings.Contains(out, "d:Close") {
		t.Fatalf("want n and d hints for 2/10 tabs, got:\n%s", out)
	}

	// Ten of ten tabs: the n hint is gone, d remains.
	out = New(rows, 0, 120, 40, 10, 10).View(120, 40)
	if strings.Contains(out, "n:New") || !strings.Contains(out, "d:Close") {
		t.Fatalf("want no n hint at the tab cap, d still shown, got:\n%s", out)
	}

	// A single tab: the d hint is gone, n remains.
	single := []Row{{Number: 1, Active: true}}
	out = New(single, 0, 120, 40, 1, 10).View(120, 40)
	if strings.Contains(out, " d:") || !strings.Contains(out, "n") {
		t.Fatalf("want no d hint for a single tab, n still shown, got:\n%s", out)
	}
}

func TestFooterActionGuards(t *testing.T) {
	// At the tab cap, n does nothing even though it is not advertised.
	rows := []Row{{Number: 1, Active: true}, {Number: 2}}
	m := New(rows, 0, 100, 30, 10, 10)
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}); cmd != nil {
		t.Fatalf("want n ignored at the tab cap, got %v", cmd())
	}

	// With one tab, d does nothing even though it is not advertised.
	single := []Row{{Number: 1, Active: true}}
	m = New(single, 0, 100, 30, 1, 10)
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}); cmd != nil {
		t.Fatalf("want d ignored for a single tab, got %v", cmd())
	}
}

func TestViewRenders(t *testing.T) {
	rows := []Row{
		{Number: 1, Left: "/a/very/long/left/path/here", Right: "/b/very/long/right/path/here", Active: true},
		{Number: 2, Left: "/x", Right: "/y"},
	}
	m := New(rows, 0, 120, 40, 2, 10)
	out := m.View(120, 40)
	if !strings.Contains(out, "Tabs") {
		t.Fatalf("want the box titled Tabs, got:\n%s", out)
	}
	// Both long paths should be clipped to fit, not overflow.
	for _, line := range strings.Split(out, "\n") {
		if ansiWidth := len([]rune(line)); ansiWidth > 120 {
			t.Fatalf("line overflows the 120-cell width:\n%s", line)
		}
	}
}
