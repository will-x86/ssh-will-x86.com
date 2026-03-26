package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/will-x86/ssh-will-x86/pkg/content"
)

const (
	HeaderHeight = 1
	FooterHeight = 1
)

type Model struct {
	term        string
	State       State
	profile     string
	width       int
	height      int
	bg          string
	TxtStyle    lipgloss.Style
	QuitStyle   lipgloss.Style
	HeaderStyle lipgloss.Style

	viewport viewport.Model
	content  string
	tooLong  bool

	projectsPosts  []content.Project
	selectedPost   *content.Project
	inProjectsList bool
	projectsList   list.Model

	messageInput textarea.Model
	nameInput    textinput.Model
	username     string
	editingName  bool
	messageSent  bool
}

// Creates model per ssh session
func NewTeaHandler() func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, _ := s.Pty()
		contentHeight := pty.Window.Height - HeaderHeight - FooterHeight
		renderer := bubbletea.MakeRenderer(s)

		txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))
		quitStyle := renderer.NewStyle().Foreground(lipgloss.Color("15"))
		headerStyle := renderer.NewStyle().Bold(true).Background(lipgloss.Color("62")).PaddingLeft(2)

		projectsPosts, err := content.LoadProjects()
		if err != nil {
			log.Error("Failed to load projects", "error", err)
			projectsPosts = []content.Project{}
		}

		items := make([]list.Item, len(projectsPosts))
		for i, post := range projectsPosts {
			items[i] = post
		}
		delegate := list.NewDefaultDelegate()
		projectsList := list.New(items, delegate, pty.Window.Width, contentHeight-2)
		projectsList.SetShowHelp(false)
		projectsList.SetShowTitle(false)
		projectsList.SetFilteringEnabled(false)
		projectsList.Styles.PaginationStyle = lipgloss.NewStyle()

		bg := "light"
		if renderer.HasDarkBackground() {
			bg = "dark"
		}

		vp := viewport.New(pty.Window.Width, contentHeight)
		vp.Style = renderer.NewStyle().Border(lipgloss.RoundedBorder())

		ta := textarea.New()
		ta.Placeholder = "Type your message here..."
		ta.Focus()
		ta.SetWidth(pty.Window.Width - 4)
		ta.SetHeight(5)

		nameInput := textinput.New()
		nameInput.Placeholder = "Your name"
		nameInput.Width = 30

		username := s.User()
		if username == "" {
			username = "anonymous"
		}

		m := Model{
			term:           pty.Term,
			profile:        renderer.ColorProfile().Name(),
			width:          pty.Window.Width,
			height:         pty.Window.Height,
			bg:             bg,
			TxtStyle:       txtStyle,
			QuitStyle:      quitStyle,
			HeaderStyle:    headerStyle,
			viewport:       vp,
			content:        "",
			projectsPosts:  projectsPosts,
			inProjectsList: true,
			projectsList:   projectsList,
			messageInput:   ta,
			nameInput:      nameInput,
			username:       username,
			editingName:    false,
		}
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}
