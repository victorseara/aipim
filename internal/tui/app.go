package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

// AppAction describes the action chosen from the main TUI.
type AppAction int

const (
	// ActionQuit exits the TUI without doing more work.
	ActionQuit AppAction = iota
	// ActionLaunch launches the selected profile.
	ActionLaunch
	// ActionCreate opens the profile creation flow.
	ActionCreate
	// ActionEdit opens the profile edit flow for the selected profile.
	ActionEdit
	// ActionDelete removes the selected profile.
	ActionDelete
	// ActionSettings opens the settings flow.
	ActionSettings
)

// AppResult captures the result of a TUI interaction.
type AppResult struct {
	Action      AppAction
	ProfileName string
}

type profileItem struct {
	profile   profile.Profile
	agentName string
}

func (i profileItem) FilterValue() string {
	return strings.Join([]string{i.profile.Name, i.profile.Path, i.agentName}, " ")
}

type profileDelegate struct{}

func (d profileDelegate) Height() int  { return 1 }
func (d profileDelegate) Spacing() int { return 0 }
func (d profileDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d profileDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	profileItem, ok := item.(profileItem)
	if !ok {
		return
	}

	bullet := "  "
	style := profileItemStyle
	if index == m.Index() {
		bullet = "● "
		style = selectedItemStyle
	}

	pathDisplay := shortenHome(profileItem.profile.Path)
	if strings.TrimSpace(pathDisplay) == "" {
		pathDisplay = "(agent-managed)"
	}
	nameDisplay := profileItem.profile.Name
	if alias := strings.TrimSpace(profileItem.profile.Alias); alias != "" {
		nameDisplay = fmt.Sprintf("%s (%s)", profileItem.profile.Name, alias)
	}
	line := fmt.Sprintf("%s%-18s %-12s %s", bullet, nameDisplay, profileItem.agentName, pathDisplay)
	fmt.Fprint(w, style.Render(line))
}

type model struct {
	list     list.Model
	width    int
	height   int
	result   AppResult
	showHelp bool
}

func newModel(cfg *config.AppConfig) model {
	items := make([]list.Item, 0, len(cfg.Profiles))
	for _, configuredProfile := range cfg.Profiles {
		agentName := configuredProfile.AgentName
		if strings.TrimSpace(agentName) == "" {
			agentName = cfg.DefaultAgentName
		}
		if strings.TrimSpace(agentName) == "" {
			agentName = "unassigned"
		}

		items = append(items, profileItem{
			profile:   configuredProfile,
			agentName: strings.ToLower(agentName),
		})
	}

	profileList := list.New(items, profileDelegate{}, 0, 0)
	profileList.SetShowTitle(false)
	profileList.SetShowStatusBar(false)
	profileList.SetShowHelp(false)
	profileList.SetFilteringEnabled(false)
	profileList.Styles.PaginationStyle = mutedStyle

	return model{
		list:   profileList,
		width:  80,
		height: 24,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-6)
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
			case "ctrl+c":
				m.result.Action = ActionQuit
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "?":
			m.showHelp = true
			return m, nil
		case "q", "ctrl+c":
			m.result.Action = ActionQuit
			return m, tea.Quit
		case "n":
			m.result.Action = ActionCreate
			return m, tea.Quit
		case "s":
			m.result.Action = ActionSettings
			return m, tea.Quit
		case "e":
			if len(m.list.Items()) == 0 {
				return m, nil
			}

			selected := m.list.SelectedItem().(profileItem)
			m.result = AppResult{Action: ActionEdit, ProfileName: selected.profile.Name}
			return m, tea.Quit
		case "d":
			if len(m.list.Items()) == 0 {
				return m, nil
			}

			selected := m.list.SelectedItem().(profileItem)
			m.result = AppResult{Action: ActionDelete, ProfileName: selected.profile.Name}
			return m, tea.Quit
		case "enter":
			if len(m.list.Items()) == 0 {
				return m, nil
			}

			selected := m.list.SelectedItem().(profileItem)
			m.result = AppResult{Action: ActionLaunch, ProfileName: selected.profile.Name}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := titleStyle.Render("aipim")

	if m.showHelp {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			helpOverlay(),
			helpTextStyle.Render("press ? or esc to close"),
		)
	}

	footer := helpTextStyle.Render(footerLegend(m.width))

	if len(m.list.Items()) == 0 {
		empty := panelStyle.Render("No profiles yet. Press n to create one.")
		return lipgloss.JoinVertical(lipgloss.Left, header, empty, footer)
	}

	selectedDesc := ""
	if selected, ok := m.list.SelectedItem().(profileItem); ok {
		if d := strings.TrimSpace(selected.profile.Description); d != "" {
			first := d
			if idx := strings.IndexByte(first, '\n'); idx >= 0 {
				first = first[:idx]
			}
			max := m.width - 4
			if max > 0 && len(first) > max {
				first = first[:max-1] + "…"
			}
			selectedDesc = helpTextStyle.Render("» " + first)
		}
	}

	parts := []string{header, m.list.View()}
	if selectedDesc != "" {
		parts = append(parts, selectedDesc)
	}
	parts = append(parts,
		lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", max(48, m.width-2))),
		footer,
	)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// footerLegend returns a width-aware key legend.
// On narrow terminals it collapses to just `? for help`.
func footerLegend(width int) string {
	full := "enter launch  n new  e edit  d delete  s settings  ? help  q quit"
	if width > 0 && width < len(full)+2 {
		return "? help  q quit"
	}
	return full
}

// helpOverlay returns the full keymap, rendered inside a styled panel.
func helpOverlay() string {
	rows := []struct{ key, desc string }{
		{"enter", "launch the highlighted profile"},
		{"n", "create a new profile"},
		{"e", "edit the highlighted profile"},
		{"d", "delete the highlighted profile"},
		{"s", "open settings (agents, default agent)"},
		{"↑/↓", "move the selection"},
		{"?", "toggle this help overlay"},
		{"q / ctrl+c", "quit"},
	}
	var b strings.Builder
	b.WriteString("Keyboard shortcuts\n\n")
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", r.key, r.desc))
	}
	b.WriteString("\nFor scriptable usage, run `aipim --help` from the shell.")
	return panelStyle.Render(b.String())
}

// RunApp runs the main profile list TUI until the user quits or launches a profile.
func RunApp(cfg *config.AppConfig) (AppResult, error) {
	for {
		program := tea.NewProgram(newModel(cfg), tea.WithAltScreen())
		finalModel, err := program.Run()
		if err != nil {
			return AppResult{}, err
		}

		final := finalModel.(model)
		switch final.result.Action {
		case ActionCreate:
			if _, err := RunCreateFlow(cfg); err != nil {
				return AppResult{}, err
			}
		case ActionEdit:
			if _, err := RunEditFlow(cfg, final.result.ProfileName); err != nil {
				return AppResult{}, err
			}
		case ActionDelete:
			if err := deleteProfile(cfg, final.result.ProfileName); err != nil {
				return AppResult{}, err
			}
		case ActionSettings:
			if err := RunSettingsFlow(cfg); err != nil {
				return AppResult{}, err
			}
		default:
			return final.result, nil
		}
	}
}

func deleteProfile(cfg *config.AppConfig, name string) error {
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Delete profile").
				Description(dangerStyle.Render(fmt.Sprintf("Delete profile %q?", name))).
				Affirmative("Delete").
				Negative("Cancel").
				Value(&confirmed),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	updatedProfiles, err := profile.DeleteByName(cfg.Profiles, name)
	if err != nil {
		return err
	}
	cfg.Profiles = updatedProfiles
	return cfg.Save()
}

func shortenHome(path string) string {
	prefix := userHomePrefix()
	if prefix == "" || !strings.HasPrefix(path, prefix) {
		return strings.TrimSpace(path)
	}

	return strings.TrimSpace(strings.Replace(path, prefix, "~/", 1))
}

func userHomePrefix() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return home + "/"
}

func max(left, right int) int {
	if left > right {
		return left
	}

	return right
}
