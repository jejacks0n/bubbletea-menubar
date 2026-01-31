# Bubble Tea Menu Bar

A reusable menu bar component for [Bubble Tea](https://github.com/charmbracelet/bubbletea) applications, featuring recursive dropdowns, keyboard navigation, and customizable styling via [Lip Gloss](https://github.com/charmbracelet/lipgloss).

![Bubble Tea Menu Bar Example](https://github.com/user-attachments/assets/dce5ee8e-f932-4bfe-8663-88b117adb5bc)

## Features

- **Top-level horizontal menu bar**
- **Recursive dropdown submenus**
- **Focus Management**: Toggle focus on/off (e.g., with `Esc`).
- **Keyboard Navigation**: Arrow keys, Enter, Esc.
- **Hotkey Support**: Jump to (or execute the action) of items using hotkey characters.
- **Shortcuts Display**: Visual hints for global shortcuts (e.g., `Ctrl+C` / `⌃+C`).
- **Smart Overlay**: Render dropdowns over your content without clearing the background, preserving text and colors underneath.
- **Customizable Styling**: Full control over colors, borders, and padding via `Lip Gloss`.

## Installation

```bash
go get github.com/jejacks0n/bubbletea-menubar
```

## Usage

### Define Menu Items
Create a hierarchy of `MenuItem`s. Each item can have an action or a submenu.

```go
fileMenu := []menubar.MenuItem{
    {Label: "New", Hotkey: "N", Shortcut: "Ctrl+N"},
    {Label: "Open", Hotkey: "O", Shortcut: "Ctrl+O"},
    {Label: "Save", Hotkey: "S", Shortcut: "Ctrl+S", Disabled: true},
    menubar.Separator(),
    {Label: "Exit", Hotkey: "x", Action: func() tea.Msg { return tea.Quit() }},
}

items := []menubar.MenuItem{
    {Label: "File", Hotkey: "F", SubMenu: fileMenu},
    menubar.Separator(),
    {Label: "Help", Hotkey: "H", SubMenu: []menubar.MenuItem{{Label: "About"}}},
}
```

### Initialize Model
Initialize the model. You can set it to start unfocused (`Active = false`) if you want the user to explicitly activate it (e.g., by pressing `Esc`).

```go
m := menubar.New(items)
m.Active = false // Start unfocused
```

### Update Loop
Handle messages and delegate to the menubar. You can also implement logic to toggle focus.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "esc" {
            // Toggle focus logic
            if !m.menubar.Active {
                m.menubar.Active = true
                return m, nil
            }
            // If active and no submenu open, unfocus
            if m.menubar.OpenSubMenu == -1 {
                m.menubar.Active = false
                return m, nil
            }
        }
    }

    var cmd tea.Cmd
    m.menubar, cmd = m.menubar.Update(msg)
    return m, cmd
}
```

### View & Overlay
To correctly overlay dropdowns on top of your content without erasing the background, use `ViewDropdownLayers` and the `Overlay` helper.

```go
func (m model) View() string {
    // Render the menubar
    bar := m.menubar.ViewBarWithRightSide("Status", m.width)
    
    // Render your main content
    content := "My Application Content..."
    fullView := lipgloss.JoinVertical(lipgloss.Top, bar, content)

    // Overlay dropdown layers
    // ViewDropdownLayers returns a list of menu parts (layers) and their positions.
    if layers, x := m.menubar.ViewDropdownLayers(); len(layers) > 0 {
        for _, layer := range layers {
            // x+layer.X is the absolute X position
            // 1+layer.Y is the absolute Y position (1 accounts for the bar height)
            fullView = menubar.Overlay(fullView, layer.Content, x+layer.X, 1+layer.Y)
        }
    }

    return fullView
}
```

## Styling

You can customize the appearance by modifying the `Styles` field of the `menubar.Model`.

```go
m.Styles.Bar = m.Styles.Bar.Background(lipgloss.Color("#333"))
m.Styles.SelectedItem = m.Styles.SelectedItem.Background(lipgloss.Color("#8800CC"))

// Use Rounded Borders for Dropdowns
m.Styles.Dropdown = m.Styles.Dropdown.Border(lipgloss.RoundedBorder())
```

## License

This library is released under the MIT license:

* https://opensource.org/licenses/MIT

Copyright 2026 [jejacks0n](https://github.com/jejacks0n)

## Make Code Not War
