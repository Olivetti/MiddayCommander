package theme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

func TestLoadByName(t *testing.T) {
	// A temp config dir, so LoadByName finds the file (CI has no
	// ~/.config/mdc/themes).
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	themesDir := filepath.Join(tmp, "mdc", "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "themes", "catppuccin-mocha.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, "catppuccin-mocha.toml"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	th, err := LoadByName("catppuccin-mocha")
	if err != nil {
		t.Fatalf("LoadByName error: %v", err)
	}

	def := Default()

	// Not the default (ANSI 15 on 4).
	if th.FileNormal.GetForeground() == def.FileNormal.GetForeground() &&
		th.FileNormal.GetBackground() == def.FileNormal.GetBackground() {
		t.Errorf("FileNormal was not overridden by theme")
	}

	t.Logf("FileNormal fg=%v bg=%v", th.FileNormal.GetForeground(), th.FileNormal.GetBackground())
	t.Logf("FileDir fg=%v bg=%v bold=%v", th.FileDir.GetForeground(), th.FileDir.GetBackground(), th.FileDir.GetBold())
	t.Logf("PanelBorder fg=%v bg=%v", th.PanelBorder.GetForeground(), th.PanelBorder.GetBackground())
	t.Logf("StatusBar fg=%v bg=%v", th.StatusBar.GetForeground(), th.StatusBar.GetBackground())
}

func TestTabSectionParses(t *testing.T) {
	// The [tab] keys must actually parse.
	var tf ThemeFile
	if _, err := toml.Decode(`
[tab]
fg        = "#111111"
bg        = "#222222"
active_fg = "#333333"
active_bg = "#444444"
`, &tf); err != nil {
		t.Fatal(err)
	}
	if tf.Tab.FG != "#111111" || tf.Tab.BG != "#222222" ||
		tf.Tab.ActiveFG != "#333333" || tf.Tab.ActiveBG != "#444444" {
		t.Fatalf("documented [tab] keys were not parsed: %+v", tf.Tab)
	}
}

func TestTabBarFollowsTheme(t *testing.T) {
	mocha := mochaTheme()
	classic := Default()

	// With no [tab] section, the bar falls back to the F-key styles.
	if mocha.Tab.GetForeground() == classic.Tab.GetForeground() &&
		mocha.Tab.GetBackground() == classic.Tab.GetBackground() {
		t.Error("mocha's tab bar should not use the classic palette")
	}
	if mocha.Tab.GetForeground() != mocha.FKeyHint.GetForeground() ||
		mocha.Tab.GetBackground() != mocha.FKeyHint.GetBackground() {
		t.Error("an unconfigured [tab] should fall back to the theme's key hint style")
	}
	if mocha.TabActive.GetForeground() != mocha.StatusBar.GetForeground() ||
		mocha.TabActive.GetBackground() != mocha.StatusBar.GetBackground() {
		t.Error("an unconfigured [tab] should fall back to the theme's status bar style")
	}
}

func TestOrDefault(t *testing.T) {
	def := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	empty := lipgloss.NewStyle()
	colored := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))

	result := orDefault(empty, def)
	if result.GetForeground() != def.GetForeground() {
		t.Errorf("orDefault should return default for empty style")
	}

	result = orDefault(colored, def)
	if result.GetForeground() == def.GetForeground() {
		t.Errorf("orDefault should return colored style, not default")
	}
}
