package menubar

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type MenuItem struct {
	Label       string
	Hotkey      string
	Shortcut    string
	Action      func() tea.Msg
	SubMenu     []MenuItem
	IsSeparator bool
	Disabled    bool
}

func Separator() MenuItem {
	return MenuItem{IsSeparator: true}
}

type Model struct {
	Items        []MenuItem
	Active       bool
	Selection    int
	OpenSubMenu  int    // Index of the open submenu, -1 if none
	SubMenuState *Model // The model for the open submenu (recursive)

	// Styling
	Styles Styles

	// Configuration
	isDropdown bool // True if this model represents a dropdown menu
}

type Styles struct {
	Bar              lipgloss.Style
	Item             lipgloss.Style
	SelectedItem     lipgloss.Style
	Shortcut         lipgloss.Style
	Dropdown         lipgloss.Style
	DropdownItem     lipgloss.Style
	DropdownSelected lipgloss.Style
	ShortcutSelected lipgloss.Style
	Hotkey           lipgloss.Style
	Separator        lipgloss.Style
	Disabled         lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Bar: lipgloss.NewStyle().
			Background(lipgloss.Color("#5F00FF")).
			Foreground(lipgloss.Color("#FFFFFF")),
		Item: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("#5F00FF")).
			Foreground(lipgloss.Color("#FFFFFF")),
		SelectedItem: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("#FF5FAF")).
			Foreground(lipgloss.Color("#FFFFFF")),
		Shortcut: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")),
		Dropdown: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#5F5FD7")),
		DropdownItem: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#CCCCCC")),
		DropdownSelected: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("#666666")),
		ShortcutSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111111")),
		Hotkey: lipgloss.NewStyle().
			Underline(true),
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")),
		Disabled: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#666666")),
	}
}

func New(items []MenuItem) Model {
	return Model{
		Items:       items,
		Styles:      DefaultStyles(),
		OpenSubMenu: -1,
		Selection:   0,
		Active:      true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// Handle mouse always to allow activation on click
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouse(mouseMsg)
	}

	if !m.Active {
		return m, nil
	}

	// Handle navigation when a submenu is open
	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		// We need to intercept Left/Right for top-level navigation if we are the top bar
		if !m.isDropdown {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch msg.String() {
				case "left":
					if !m.SubMenuState.hasOpenSubmenu() {
						m.Selection--
						if m.Selection < 0 {
							m.Selection = len(m.Items) - 1
						}
						// Skip separators and disabled items
						start := m.Selection
						for m.Items[m.Selection].IsSeparator || m.Items[m.Selection].Disabled {
							m.Selection--
							if m.Selection < 0 {
								m.Selection = len(m.Items) - 1
							}
							if m.Selection == start {
								break
							}
						}
						m.openCurrentSelection()
						return m, nil
					}
				case "right":
					if !m.SubMenuState.wantsToHandleRight() {
						m.Selection++
						if m.Selection >= len(m.Items) {
							m.Selection = 0
						}
						// Skip separators and disabled items
						start := m.Selection
						for m.Items[m.Selection].IsSeparator || m.Items[m.Selection].Disabled {
							m.Selection++
							if m.Selection >= len(m.Items) {
								m.Selection = 0
							}
							if m.Selection == start {
								break
							}
						}
						m.openCurrentSelection()
						return m, nil
					}
				}
			}
		}

		// Delegate to submenu
		newSubModel, cmd := m.SubMenuState.Update(msg)
		m.SubMenuState = &newSubModel

		// Check if submenu closed itself (e.g. via Esc or Left in dropdown)
		if !m.SubMenuState.Active {
			m.OpenSubMenu = -1
			m.SubMenuState = nil
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Check for hotkeys
		// 1. Exact match (case-sensitive)
		for i, item := range m.Items {
			if item.IsSeparator || item.Disabled {
				continue
			}
			if item.Hotkey != "" && key == item.Hotkey {
				m.Selection = i
				if len(item.SubMenu) > 0 {
					m.openCurrentSelection()
				} else if item.Action != nil {
					return m, func() tea.Msg { return item.Action() }
				}
				return m, nil
			}
		}
		// 2. Fallback to case-insensitive match
		for i, item := range m.Items {
			if item.IsSeparator || item.Disabled {
				continue
			}
			if item.Hotkey != "" && strings.EqualFold(key, item.Hotkey) {
				m.Selection = i
				if len(item.SubMenu) > 0 {
					m.openCurrentSelection()
				} else if item.Action != nil {
					return m, func() tea.Msg { return item.Action() }
				}
				return m, nil
			}
		}

		switch key {
		case "left":
			if m.isDropdown {
				// Close this dropdown
				m.Active = false
				return m, nil
			}
			m.Selection--
			if m.Selection < 0 {
				m.Selection = len(m.Items) - 1
			}
			// Skip separators and disabled items
			start := m.Selection
			for m.Items[m.Selection].IsSeparator || m.Items[m.Selection].Disabled {
				m.Selection--
				if m.Selection < 0 {
					m.Selection = len(m.Items) - 1
				}
				if m.Selection == start {
					break
				}
			}
		case "right":
			if m.isDropdown {
				// If current item has submenu, open it
				item := m.Items[m.Selection]
				if len(item.SubMenu) > 0 {
					m.openCurrentSelection()
				}
			} else {
				m.Selection++
				if m.Selection >= len(m.Items) {
					m.Selection = 0
				}
				// Skip separators and disabled items
				start := m.Selection
				for m.Items[m.Selection].IsSeparator || m.Items[m.Selection].Disabled {
					m.Selection++
					if m.Selection >= len(m.Items) {
						m.Selection = 0
					}
					if m.Selection == start {
						break
					}
				}
			}
		case "up":
			if m.isDropdown {
				m.Selection--
				if m.Selection < 0 {
					m.Selection = len(m.Items) - 1
				}
				// Skip separators and disabled items
				start := m.Selection
				for m.Items[m.Selection].IsSeparator || m.Items[m.Selection].Disabled {
					m.Selection--
					if m.Selection < 0 {
						m.Selection = len(m.Items) - 1
					}
					if m.Selection == start {
						break
					}
				}
			}
		case "down":
			if m.isDropdown {
				m.Selection++
				if m.Selection >= len(m.Items) {
					m.Selection = 0
				}
				// Skip separators and disabled items
				start := m.Selection
				for m.Items[m.Selection].IsSeparator || m.Items[m.Selection].Disabled {
					m.Selection++
					if m.Selection >= len(m.Items) {
						m.Selection = 0
					}
					if m.Selection == start {
						break
					}
				}
			} else {
				// Open menu
				if len(m.Items) > 0 {
					m.openCurrentSelection()
				}
			}
		case "enter":
			if len(m.Items) > 0 {
				item := m.Items[m.Selection]
				if item.Disabled {
					return m, nil
				}
				if len(item.SubMenu) > 0 {
					m.openCurrentSelection()
				} else if item.Action != nil {
					return m, func() tea.Msg { return item.Action() }
				}
			}
		case "esc":
			if m.isDropdown {
				m.Active = false
			} else {
				m.OpenSubMenu = -1
				m.SubMenuState = nil
			}
		}
	}

	return m, nil
}

func (m *Model) openCurrentSelection() {
	item := m.Items[m.Selection]
	if len(item.SubMenu) > 0 {
		m.OpenSubMenu = m.Selection
		sub := New(item.SubMenu)
		sub.isDropdown = true
		sub.Styles = m.Styles
		m.SubMenuState = &sub
	}
}

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	handled, cmd := m.checkMouse(msg, 0, 0)

	// If click outside, close menus
	if !handled && msg.Type == tea.MouseRelease {
		m.Active = false
		m.OpenSubMenu = -1
		m.SubMenuState = nil
	}

	return m, cmd
}

// checkMouse performs hit testing. Returns true if the event was handled (hit something).
func (m *Model) checkMouse(msg tea.MouseMsg, baseX, baseY int) (bool, tea.Cmd) {
	// 1. Check open submenu first (it's on top)
	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		var subX, subY int
		if m.isDropdown {
			// Submenu of a dropdown
			// Position is to the right of the rendering
			width, _ := m.getDropdownDimensions()
			subX = baseX + width
			topBorder := lipgloss.Height(m.Styles.Dropdown.GetBorderStyle().Top)

			yOffset := topBorder
			for i := 0; i < m.OpenSubMenu; i++ {
				h := lipgloss.Height(m.Styles.DropdownItem.Render("A"))
				if m.Items[i].IsSeparator {
					h = lipgloss.Height(m.Styles.Separator.Render("-"))
				}
				yOffset += h
			}
			subY = baseY + yOffset
		} else {
			// Submenu of the bar
			// X = offset of item
			// Y = 1
			subX = baseX
			for i := 0; i < m.OpenSubMenu; i++ {
				subX += m.measureItem(i)
			}
			subY = baseY + lipgloss.Height(m.Styles.Bar.Render("A")) // Height of bar
		}

		handled, cmd := m.SubMenuState.checkMouse(msg, subX, subY)
		if handled {
			return true, cmd
		}
	}

	// 2. Check ourselves
	if m.isDropdown {
		// Hit test this dropdown
		width, height := m.getDropdownDimensions()
		if msg.X >= baseX && msg.X < baseX+width && msg.Y >= baseY && msg.Y < baseY+height {
			// Hit!
			// Calculate Item Index
			topBorder := lipgloss.Height(m.Styles.Dropdown.GetBorderStyle().Top)
			localY := msg.Y - baseY - topBorder

			// We iterate items to find which one covers localY
			currentY := 0
			for i := range m.Items {
				itemH := lipgloss.Height(m.Styles.DropdownItem.Render("A"))
				if m.Items[i].IsSeparator {
					itemH = lipgloss.Height(m.Styles.Separator.Render("-"))
				}

				if localY >= currentY && localY < currentY+itemH {
					if m.Items[i].IsSeparator || m.Items[i].Disabled {
						return true, nil
					}
					m.Selection = i

					if msg.Type == tea.MouseRelease {
						if len(m.Items[i].SubMenu) > 0 {
							m.openCurrentSelection()
						} else if m.Items[i].Action != nil {
							return true, func() tea.Msg { return m.Items[i].Action() }
						}
					} else if msg.Type == tea.MouseMotion {
						if m.OpenSubMenu != -1 && m.OpenSubMenu != i {
							m.OpenSubMenu = -1
							m.SubMenuState = nil
						}
					}
					return true, nil
				}
				currentY += itemH
			}
			return true, nil
		}
	} else {
		barHeight := lipgloss.Height(m.Styles.Bar.Render("A"))
		if msg.Y >= baseY && msg.Y < baseY+barHeight {
			currentX := baseX
			for i := range m.Items {
				w := m.measureItem(i)
				if msg.X >= currentX && msg.X < currentX+w {
					m.Selection = i

					if msg.Type == tea.MouseRelease {
						if !m.Active {
							m.Active = true
						}
						if len(m.Items[i].SubMenu) > 0 {
							if m.OpenSubMenu == i {
								m.OpenSubMenu = -1
								m.SubMenuState = nil
							} else {
								m.openCurrentSelection()
							}
						} else if m.Items[i].Action != nil {
							return true, func() tea.Msg { return m.Items[i].Action() }
						}
					} else if msg.Type == tea.MouseMotion {
						if m.Active && m.OpenSubMenu != -1 && m.OpenSubMenu != i {
							m.openCurrentSelection()
						}
					}
					return true, nil
				}
				currentX += w
			}
			return true, nil
		}
	}

	return false, nil
}

func (m Model) hasOpenSubmenu() bool {
	return m.OpenSubMenu != -1 && m.SubMenuState != nil
}

func (m Model) wantsToHandleRight() bool {
	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		return m.SubMenuState.wantsToHandleRight()
	}
	return len(m.Items) > 0 && len(m.Items[m.Selection].SubMenu) > 0
}

func (m Model) View() string {
	if m.isDropdown {
		return m.viewDropdown()
	}
	return m.ViewWithRightSide("", 0)
}

func (m Model) ViewWithRightSide(right string, width int) string {
	if m.isDropdown {
		return m.viewDropdown()
	}
	bar := m.renderBarContent(right, width)
	dropdown, offset := m.ViewDropdown()

	if dropdown != "" {
		return lipgloss.JoinVertical(lipgloss.Top, bar, lipgloss.NewStyle().MarginLeft(offset).Render(dropdown))
	}
	return bar
}

func (m Model) ViewBar() string {
	if m.isDropdown {
		return ""
	}
	return m.renderBarContent("", 0)
}

func (m Model) ViewBarWithRightSide(right string, width int) string {
	if m.isDropdown {
		return ""
	}
	return m.renderBarContent(right, width)
}

type DropdownLayer struct {
	Content string
	X       int
	Y       int
}

func (m Model) ViewDropdown() (string, int) {
	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		dropdown := m.SubMenuState.View()
		offset := m.getDropdownOffset()
		return dropdown, offset
	}
	return "", 0
}

func (m Model) ViewDropdownLayers() ([]DropdownLayer, int) {
	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		offset := m.getDropdownOffset()
		layers := m.SubMenuState.getLayersRecursive(0, 0)
		return layers, offset
	}
	return nil, 0
}

func (m Model) getLayersRecursive(baseX, baseY int) []DropdownLayer {
	currentView := m.renderSingleDropdown()
	layers := []DropdownLayer{{Content: currentView, X: baseX, Y: baseY}}

	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		menuWidth := lipgloss.Width(currentView)
		// Assuming 1 line for top border + selection index
		yOffset := m.Selection + 1
		subLayers := m.SubMenuState.getLayersRecursive(baseX+menuWidth, baseY+yOffset)
		layers = append(layers, subLayers...)
	}
	return layers
}

func (m Model) measureItem(i int) int {
	if m.Items[i].IsSeparator {
		return lipgloss.Width(m.Styles.Item.Render("|"))
	}
	style := m.Styles.Item

	if m.Active && i == m.Selection {
		style = m.Styles.SelectedItem
	}
	baseStyle := style.Copy().UnsetPadding()
	rendered := style.Render(m.renderLabel(m.Items[i], baseStyle))
	return lipgloss.Width(rendered)
}

func (m Model) renderBarContent(right string, width int) string {
	var views []string
	for i, item := range m.Items {
		if item.IsSeparator {
			views = append(views, m.Styles.Item.Render("|"))
			continue
		}
		style := m.Styles.Item
		if m.Active && i == m.Selection {
			style = m.Styles.SelectedItem
		}
		if item.Disabled {
			style = m.Styles.Disabled.Copy().Inherit(m.Styles.Item)
		}
		baseStyle := style.Copy().UnsetPadding()
		views = append(views, style.Render(m.renderLabel(item, baseStyle)))
	}

	fillStyle := m.Styles.Bar.Copy().UnsetPadding().BorderTop(false).BorderRight(false).BorderBottom(false).BorderLeft(false).Margin(0)

	if width > 0 {
		itemsWidth := lipgloss.Width(lipgloss.JoinHorizontal(lipgloss.Top, views...))
		rightWidth := lipgloss.Width(right)
		availableWidth := width - m.Styles.Bar.GetHorizontalFrameSize()
		spacerWidth := availableWidth - itemsWidth - rightWidth

		if spacerWidth > 0 {
			views = append(views, fillStyle.Render(strings.Repeat(" ", spacerWidth)))
		}
	}

	if right != "" {
		views = append(views, fillStyle.Render(right))
	}

	return m.Styles.Bar.Render(lipgloss.JoinHorizontal(lipgloss.Top, views...))
}

func (m Model) getDropdownOffset() int {
	if m.OpenSubMenu == -1 {
		return 0
	}
	offset := 0
	for i := 0; i < m.OpenSubMenu; i++ {
		baseStyle := m.Styles.Item.Copy().UnsetPadding()
		offset += lipgloss.Width(m.Styles.Item.Render(m.renderLabel(m.Items[i], baseStyle)))
	}
	return offset
}

func (m Model) viewDropdown() string {
	menu := m.renderSingleDropdown()

	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		subMenu := m.SubMenuState.View()
		padding := strings.Repeat("\n", m.Selection+1) // +1 for top border
		return lipgloss.JoinHorizontal(lipgloss.Top, menu, padding+subMenu)
	}

	return menu
}

func (m Model) getDropdownDimensions() (int, int) {
	maxLabelWidth := 0
	maxShortcutWidth := 0
	hasSubmenu := false

	for _, item := range m.Items {
		w := lipgloss.Width(item.Label)
		if w > maxLabelWidth {
			maxLabelWidth = w
		}
		sw := lipgloss.Width(item.Shortcut)
		if sw > maxShortcutWidth {
			maxShortcutWidth = sw
		}
		if len(item.SubMenu) > 0 {
			hasSubmenu = true
		}
	}

	maxRightWidth := maxShortcutWidth
	if hasSubmenu && maxRightWidth < 2 {
		maxRightWidth = 2
	}

	dummyStyle := m.Styles.DropdownItem
	innerContentWidth := maxLabelWidth + 2 + maxRightWidth

	itemWidth := lipgloss.Width(dummyStyle.Render(strings.Repeat(" ", innerContentWidth)))
	height := len(m.Items)

	w, h := m.Styles.Dropdown.GetFrameSize()

	return itemWidth + w, height + h
}

func (m Model) renderSingleDropdown() string {
	// Calculate widths for alignment
	maxLabelWidth := 0
	maxShortcutWidth := 0
	hasSubmenu := false

	for _, item := range m.Items {
		w := lipgloss.Width(item.Label)
		if w > maxLabelWidth {
			maxLabelWidth = w
		}
		sw := lipgloss.Width(item.Shortcut)
		if sw > maxShortcutWidth {
			maxShortcutWidth = sw
		}
		if len(item.SubMenu) > 0 {
			hasSubmenu = true
		}
	}

	maxRightWidth := maxShortcutWidth
	if hasSubmenu && maxRightWidth < 2 {
		maxRightWidth = 2
	}

	// Calculate standard item width (including padding)
	innerContentWidth := maxLabelWidth + 2 + maxRightWidth
	standardWidth := lipgloss.Width(m.Styles.DropdownItem.Render(strings.Repeat(" ", innerContentWidth)))

	var views []string
	for i, item := range m.Items {
		if item.IsSeparator {
			// Calculate line length to match standardWidth when rendered with separator style
			separatorSideWidth := m.Styles.Separator.GetHorizontalFrameSize()
			lineLength := standardWidth - separatorSideWidth
			if lineLength < 0 {
				lineLength = 0
			}
			line := strings.Repeat("─", lineLength)
			views = append(views, m.Styles.Separator.Render(line))
			continue
		}

		style := m.Styles.DropdownItem
		if i == m.Selection {
			style = m.Styles.DropdownSelected
		}
		if item.Disabled {
			style = m.Styles.Disabled.Copy().Inherit(m.Styles.DropdownItem)
		}

		baseStyle := style.Copy().UnsetPadding()

		// Render Label
		label := m.renderLabel(item, baseStyle)
		currentLabelWidth := lipgloss.Width(label)

		// Pad label to max width + gap
		padding := baseStyle.Render(strings.Repeat(" ", maxLabelWidth-currentLabelWidth+2))

		// Right-side content (Shortcut or Submenu Indicator)
		rightContent := ""
		if item.Shortcut != "" {
			shortcutStyle := m.Styles.Shortcut.Copy().Inherit(baseStyle)
			if i == m.Selection {
				shortcutStyle = m.Styles.ShortcutSelected.Copy().Inherit(baseStyle)
			}
			if item.Disabled {
				shortcutStyle = m.Styles.Disabled.Copy().Inherit(baseStyle).Padding(0)
			}

			shortcutStr := shortcutStyle.Render(item.Shortcut)
			// Right align shortcut in the right column
			rightContent = baseStyle.Render(strings.Repeat(" ", maxRightWidth-lipgloss.Width(item.Shortcut))) + shortcutStr
		} else if len(item.SubMenu) > 0 {
			// Right align indicator in the right column
			rightContent = baseStyle.Render(strings.Repeat(" ", maxRightWidth-2) + " >")
		} else if maxRightWidth > 0 {
			// Empty space for items with neither
			rightContent = baseStyle.Render(strings.Repeat(" ", maxRightWidth))
		}

		// Combine: Label + Padding + RightContent
		line := label + padding + rightContent
		views = append(views, style.Render(line))
	}

	return m.Styles.Dropdown.Render(lipgloss.JoinVertical(lipgloss.Left, views...))
}

func (m Model) renderLabel(item MenuItem, baseStyle lipgloss.Style) string {
	if item.Hotkey == "" || item.Disabled {
		return baseStyle.Render(item.Label)
	}

	idx := strings.Index(item.Label, item.Hotkey)
	if idx == -1 {
		idx = strings.Index(strings.ToLower(item.Label), strings.ToLower(item.Hotkey))
	}

	if idx == -1 {
		return baseStyle.Render(item.Label)
	}

	pre := item.Label[:idx]
	hot := item.Label[idx : idx+len(item.Hotkey)]
	post := item.Label[idx+len(item.Hotkey):]

	hotStyle := m.Styles.Hotkey.Copy().Inherit(baseStyle)

	var postRendered string
	if post != "" {
		postRendered = baseStyle.Inline(true).Render(post)
	}

	return baseStyle.Render(pre) + hotStyle.Render(hot) + postRendered
}

func Overlay(bg string, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fgLine := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			bgLines = append(bgLines, strings.Repeat(" ", x)+fgLine)
			continue
		}

		bgLine := bgLines[row]
		bgWidth := lipgloss.Width(bgLine)

		if bgWidth < x {
			padding := strings.Repeat(" ", x-bgWidth)
			bgLines[row] = bgLine + padding + fgLine
			continue
		}

		prefix, _ := splitWithANSI(bgLine, x)

		fgWidth := lipgloss.Width(fgLine)
		suffixStart := x + fgWidth

		preSuffix, suffix := splitWithANSI(bgLine, suffixStart)

		ansiCodes := strings.Join(ansiRegex.FindAllString(preSuffix, -1), "")
		suffix = ansiCodes + suffix

		prefixWidth := lipgloss.Width(prefix)
		padding := ""
		if prefixWidth < x {
			padding = strings.Repeat(" ", x-prefixWidth)
		}

		bgLines[row] = prefix + padding + fgLine + suffix
	}
	return strings.Join(bgLines, "\n")
}

func splitWithANSI(s string, width int) (string, string) {
	prevI := 0
	for i := range s {
		w := lipgloss.Width(s[:i])
		if w == width {
			return s[:i], s[i:]
		}
		if w > width {
			// We exceeded the target width (e.g., wide character).
			// We split before this character.
			return s[:prevI], s[prevI:]
		}
		prevI = i
	}

	if lipgloss.Width(s) <= width {
		return s, ""
	}

	// Should not be reachable if loop covers everything, but safety fallback
	return s, ""
}
