func (m Model) leftEntries() []entry {
	k := m.keys
	return []entry{
		{"", ""},
		{"── Navigation ──", ""},
		{"Move up", fmtKeys(k.Up)},
		{"Move down", fmtKeys(k.Down)},
		{"Page up", fmtKeys(k.PageUp)},
		{"Page down", fmtKeys(k.PageDown)},
		{"Go to top", fmtKeys(k.Home)},
		{"Go to bottom", fmtKeys(k.End)},
		{"Go back", fmtKeys(k.GoBack)},
		{"Go forward", fmtKeys(k.GoForward)},
		{"Go to path", fmtKeys(k.GoTo)},
		{"Switch panel", fmtKeys(k.TogglePanel)},
		{"Swap panels", fmtKeys(k.SwapPanels)},
		{"Same dir", fmtKeys(k.SameDir)},
		{"", ""},
		{"── Selection ──", ""},
		{"Toggle select", fmtKeys(k.ToggleSelect)},
		{"Select up", fmtKeys(k.SelectUp)},
		{"Select down", fmtKeys(k.SelectDown)},
		{"Select group", fmtKeys(k.SelectGroup)},
		{"Deselect group", fmtKeys(k.DeselectGroup)},
		{"Invert selection", fmtKeys(k.InvertSelection)},
	}
}

