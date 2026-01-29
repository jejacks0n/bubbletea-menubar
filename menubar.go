package menubar

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuItem struct {
	Label    string
	Hotkey   string
	Shortcut string
	Action   func() tea.Msg
	SubMenu  []MenuItem
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
			//Background(lipgloss.Color("#FF0000")).
			BorderForeground(lipgloss.Color("#5F5FD7")),
		DropdownItem: lipgloss.NewStyle().
			Padding(0, 1).
			//Background(lipgloss.Color("#444")).
			Foreground(lipgloss.Color("#CCCCCC")),
		DropdownSelected: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("#666666")),
		ShortcutSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111111")),
		Hotkey: lipgloss.NewStyle().
			//Foreground(lipgloss.Color("#FCD200")).
			Underline(true),
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
						m.openCurrentSelection()
						return m, nil
					}
				case "right":
					if !m.SubMenuState.wantsToHandleRight() {
						m.Selection++
						if m.Selection >= len(m.Items) {
							m.Selection = 0
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
			}
		case "up":
			if m.isDropdown {
				m.Selection--
				if m.Selection < 0 {
					m.Selection = len(m.Items) - 1
				}
			}
		case "down":
			if m.isDropdown {
				m.Selection++
				if m.Selection >= len(m.Items) {
					m.Selection = 0
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
	// If not active, only a click on the bar can activate it (optional, but good UX)
	// For now, we assume if inactive, we ignore, or we can check if click is on bar.

	// We start checking from the root.
	// Root (Bar) is at 0,0 relative to this component.
	// We need to return the updated model.

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
			// Y matches the item selection
			// We need to account for border/padding of the parent dropdown
			topBorder := lipgloss.Height(m.Styles.Dropdown.GetBorderStyle().Top)
			// And item padding? Usually items are stacked.
			// The render logic puts the submenu aligned with the item.
			// Item index `m.OpenSubMenu` corresponds to Y offset.
			// Each item is usually 1 line high + vertical padding?
			// renderSingleDropdown just joins them vertically.
			// Assuming 1 line height for text, + padding.
			// Let's look at renderSingleDropdown again:
			// It joins `style.Render(line)`.
			// We need to calculate the Y offset of the *selected item*.

			yOffset := topBorder
			for i := 0; i < m.OpenSubMenu; i++ {
				yOffset += lipgloss.Height(m.Styles.DropdownItem.Render("A")) // Approx height
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

		// If clicked outside submenu, and not handled by submenu, we might close it?
		// We continue to check ourselves.
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
				// Measure height of this item
				// We can't easily measure exact height without re-rendering or assuming.
				// Assuming standard 1-line items for now (safe for menu bars usually)
				// Taking padding into account? style.Render includes padding.
				// m.Styles.DropdownItem usually has padding but it might be horizontal.
				// Vertical padding adds lines.
				itemH := lipgloss.Height(m.Styles.DropdownItem.Render("A"))

				if localY >= currentY && localY < currentY+itemH {
					// Hit item i
					m.Selection = i

					// Hover: Open submenu if exists?
					// Standard behavior: if a sibling submenu is open, switch.
					// If we are just moving mouse, we usually just highlight.

					// Click:
					if msg.Type == tea.MouseRelease {
						if len(m.Items[i].SubMenu) > 0 {
							m.openCurrentSelection()
						} else if m.Items[i].Action != nil {
							return true, func() tea.Msg { return m.Items[i].Action() }
						}
					} else if msg.Type == tea.MouseMotion {
						// Auto-switch submenu if one is already open
						// Or if we implement "hover opens"
						// For now: just highlight.
						// Note: If we had a submenu open for a DIFFERENT item, we should close it?
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
		// Top Bar Hit Test
		// Height of bar
		barHeight := lipgloss.Height(m.Styles.Bar.Render("A"))
		if msg.Y >= baseY && msg.Y < baseY+barHeight {
			// Check X
			currentX := baseX
			for i := range m.Items {
				w := m.measureItem(i)
				if msg.X >= currentX && msg.X < currentX+w {
					// Hit item i
					m.Selection = i

					if msg.Type == tea.MouseRelease {
						// Toggle or Open
						if !m.Active {
							m.Active = true
						}
						if len(m.Items[i].SubMenu) > 0 {
							// If already open, maybe close?
							if m.OpenSubMenu == i {
								// Toggle off?
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
							// If we are active and have a submenu open, switching items on hover is standard
							m.openCurrentSelection()
						}
					}
					return true, nil
				}
				currentX += w
			}
			// Hit bar but no item (spacer)
			return true, nil
		}
	}

	// If click was outside everything
	if msg.Type == tea.MouseRelease {
		// Only close if we are the top level handling this
		// But this is recursive.
		// We return false. The caller (Update) might decide what to do.
		// But since we modify m in place, we can close submenus here if we are the parent.
		// Actually, if a child didn't handle it, and we didn't handle it, it's an outside click.
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

// View returns the rendered menu bar. If a submenu is open, it is appended vertically (pushing down content).
func (m Model) View() string {
	if m.isDropdown {
		return m.viewDropdown()
	}
	return m.ViewWithRightSide("", 0)
}

// ViewWithRightSide returns the rendered menu bar with optional right-side content.
// If a submenu is open, it is appended vertically.
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

// ViewBar returns just the horizontal menu bar (without any open dropdowns).
func (m Model) ViewBar() string {
	if m.isDropdown {
		return ""
	}
	return m.renderBarContent("", 0)
}

// ViewBarWithRightSide returns just the horizontal menu bar with right-side content.
func (m Model) ViewBarWithRightSide(right string, width int) string {
	if m.isDropdown {
		return ""
	}
	return m.renderBarContent(right, width)
}

// DropdownLayer represents a single menu level to be overlaid.
type DropdownLayer struct {
	Content string
	X       int
	Y       int
}

// ViewDropdown returns the rendered dropdown (if any) and its horizontal offset relative to the bar.
// Returns "", 0 if no dropdown is open.
func (m Model) ViewDropdown() (string, int) {
	if m.OpenSubMenu != -1 && m.SubMenuState != nil {
		dropdown := m.SubMenuState.View()
		offset := m.getDropdownOffset()
		return dropdown, offset
	}
	return "", 0
}

// ViewDropdownLayers returns a list of dropdown layers (recursively) and the horizontal offset of the root dropdown.
// This allows for overlaying menus without clearing the background in a rectangular bounding box.
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
	style := m.Styles.Item
	// We simulate the selection state to get accurate width if style changes on selection
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
		style := m.Styles.Item
		if m.Active && i == m.Selection {
			style = m.Styles.SelectedItem
		}
		baseStyle := style.Copy().UnsetPadding()
		views = append(views, style.Render(m.renderLabel(item, baseStyle)))
	}

	// Prepare style for filling (spacer and right side)
	// We copy the Bar style but remove layout properties to ensure only colors apply
	fillStyle := m.Styles.Bar.Copy().UnsetPadding().BorderTop(false).BorderRight(false).BorderBottom(false).BorderLeft(false).Margin(0)

	// Calculate spacing
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
		// Add newlines to align with selection
		// We need to account for the border of the parent menu if any
		// The selection index corresponds to the item index.
		// Each item is 1 line high.
		// Plus top border (1 line).

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

	// Calculate single item width
	// Structure: Border + Padding + Label + Gap + RightContent + Padding + Border
	// We use the Style to measure padding/border
	// But styles are applied per item.
	// We can render a dummy item to measure overhead.
	dummyStyle := m.Styles.DropdownItem

	// Inner content width calculation
	// Label + Padding(Spacer) + RightContent
	// The render logic aligns them.
	// Width = maxLabelWidth + 2 (gap) + maxRightWidth
	innerContentWidth := maxLabelWidth + 2 + maxRightWidth

	// Apply item padding
	itemWidth := lipgloss.Width(dummyStyle.Render(strings.Repeat(" ", innerContentWidth)))

	// Height = number of items
	height := len(m.Items)

	// Apply Dropdown container border/padding
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

	var views []string
	for i, item := range m.Items {
		style := m.Styles.DropdownItem
		if i == m.Selection {
			style = m.Styles.DropdownSelected
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
	if item.Hotkey == "" {
		return baseStyle.Render(item.Label)
	}

	// Try exact match first
	idx := strings.Index(item.Label, item.Hotkey)
	if idx == -1 {
		// Fallback to case-insensitive match
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

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Overlay overlays the foreground string (fg) onto the background string (bg)
// at the specified x, y coordinates. This is a helper for overlaying dropdowns.
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

		// If the background line is shorter than x, pad it.
		if bgWidth < x {
			padding := strings.Repeat(" ", x-bgWidth)
			bgLines[row] = bgLine + padding + fgLine
			continue
		}

		// Robust splitting using lipgloss.Width to handle ANSI codes correctly.
		prefix, _ := splitWithANSI(bgLine, x)

		// Calculate where the suffix should start
		fgWidth := lipgloss.Width(fgLine)
		suffixStart := x + fgWidth

		preSuffix, suffix := splitWithANSI(bgLine, suffixStart)

		// Restore styles for suffix:
		// The fgLine likely ends with a reset. We need to restore the styles active
		// at the point where suffix starts. We do this by extracting all ANSI codes
		// from the part of the line before the suffix and prepending them.
		ansiCodes := strings.Join(ansiRegex.FindAllString(preSuffix, -1), "")
		suffix = ansiCodes + suffix

		// Calculate padding if prefix is shorter than x (e.g. due to wide chars being cut)
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
