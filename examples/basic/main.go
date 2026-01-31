package main

import (
	"fmt"
	"os"

	"time"

	menubar "github.com/jejacks0n/bubbletea-menubar"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	menubar  menubar.Model
	quitting bool
	width    int
	height   int
	content  string
	now      time.Time
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type actionMsg string

// //⌘❖◆✲⎈⌃⎇⌥⇧⇪⏎
func initialModel() model {
	fileMenu := []menubar.MenuItem{
		{Label: "New", Hotkey: "n", Shortcut: "⌃+N", Action: func() tea.Msg { return actionMsg("New File Created") }},
		{Label: "Open", Hotkey: "O", Shortcut: "⌃+O", Action: func() tea.Msg { return actionMsg("File Opened") }},
		{Label: "Save", Hotkey: "S", Shortcut: "⌃+S", Action: func() tea.Msg { return actionMsg("File Saved") }},
		menubar.Separator(),
		{Label: "Exit", Hotkey: "x", Shortcut: "⌃+C", Action: func() tea.Msg { return tea.Quit() }},
	}

	editMenu := []menubar.MenuItem{
		{Label: "Cut", Hotkey: "t", Shortcut: "⌃⌘+X"},
		{Label: "Copy", Hotkey: "C", Shortcut: "⌃⌘+C"},
		{Label: "Paste", Hotkey: "P", Shortcut: "⌃⌘+P"},
		menubar.Separator(),
		{
			Label:  "Find",
			Hotkey: "F",
			SubMenu: []menubar.MenuItem{
				{Label: "Find...", Hotkey: "F", Shortcut: "⌃⌘+F"},
				{Label: "Replace...", Hotkey: "R"},
				{
					Label: "Advanced",
					SubMenu: []menubar.MenuItem{
						{Label: "Regex"},
						{Label: "Case Sensitive"},
					},
				},
			},
		},
	}

	helpMenu := []menubar.MenuItem{
		{Label: "About", Hotkey: "A"},
	}

	items := []menubar.MenuItem{
		{Label: "File", Hotkey: "F", SubMenu: fileMenu},
		{Label: "Edit", Hotkey: "E", SubMenu: editMenu},
		menubar.Separator(),
		{Label: "Help", Hotkey: "H", SubMenu: helpMenu},
	}

	m := menubar.New(items)
	m.Active = false
	// Use rounded borders
	//   Options include things like: NormalBorder, RoundedBorder, BlockBorder, OuterHalfBlockBorder, InnerHalfBlockBorder, ThickBorder, DoubleBorder
	//m.Styles.Dropdown = m.Styles.Dropdown.Border(lipgloss.RoundedBorder())

	// Example content (Lorem Ipsum)
	rawContent := `
  Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
  Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. 
  Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris 
  nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in 
  reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. 
  Excepteur sint occaecat cupidatat non proident, sunt in culpa qui 
  officia deserunt mollit anim id est laborum.

  You can use the menu above to trigger actions.
  Notice how the menu dropdowns overlay this text.
`
	colors := []string{"#FF0000", "#FF7F00", "#FFFF00", "#00FF00", "#0000FF", "#4B0082", "#9400D3"}
	var content string
	i := 0
	for _, char := range rawContent {
		if char == ' ' || char == '\n' {
			content += string(char)
			continue
		}
		content += lipgloss.NewStyle().Foreground(lipgloss.Color(colors[i%len(colors)])).Render(string(char))
		i++
	}
	contentStyle := lipgloss.NewStyle().Margin(1, 2)
	styledContent := contentStyle.Render(content)

	// Custom Styling
	//m.Styles.Bar = m.Styles.Bar.Background(lipgloss.Color("#222222"))
	//m.Styles.Item = m.Styles.Item.Background(lipgloss.Color("#222222"))
	//m.Styles.SelectedItem = m.Styles.SelectedItem.
	//	Background(lipgloss.Color("#005FDF")).
	//	Foreground(lipgloss.Color("#FFFFFF")).
	//	Bold(true)

	return model{
		menubar: m,
		content: styledContent,
		now:     time.Now(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.menubar.Init(), tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "esc" {
			if !m.menubar.Active {
				m.menubar.Active = true
				return m, nil
			}
			if m.menubar.OpenSubMenu == -1 {
				m.menubar.Active = false
				return m, nil
			}
		}
	case actionMsg:
		m.content += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Render(string(msg))
	case tickMsg:
		m.now = time.Time(msg)
		return m, tick()
	}

	var cmd tea.Cmd
	m.menubar, cmd = m.menubar.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Add right-side content (Dynamic Clock)
	rightSide := lipgloss.NewStyle().Padding(0, 1).Render(m.now.Format("15:04:05"))

	// Render the bar first
	bar := m.menubar.ViewBarWithRightSide(rightSide, m.width)

	// Combine bar and content
	fullView := lipgloss.JoinVertical(lipgloss.Top, bar, m.content)

	// Overlay dropdown if open
	if layers, x := m.menubar.ViewDropdownLayers(); len(layers) > 0 {
		for _, layer := range layers {
			fullView = menubar.Overlay(fullView, layer.Content, x+layer.X, 1+layer.Y)
		}
	}

	return fullView
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
