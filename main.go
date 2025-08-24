package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"
)

const (
	headerHeight = 1
	footerHeight = 1
	homeText     = `
Intro:
Hi, I'm will-x86, and this is my personal website (sshite?).
I've been developing software since early 2018 & mainly work with Go & Rust.
More recently I've been open to other technologies, 
primarily micro-electronics and front-end (Next/React).

About myself:
- I self-host, from Ollama to Immich I love it all
- I'm a University student in the UK
- Serverless technologies fascinate me, though I go bare metal for my deployments 
- I'm starting to love designing PCB's....
`
)

var (
	hostFlag = flag.String("host", "0.0.0.0", "Host to listen on (use 0.0.0.0 for remote access)")
	portFlag = flag.String("port", "22", "Port to listen on (22 for standard SSH)")
)

func main() {
	flag.Parse()
	port := *portFlag
	host := *hostFlag
	log.Info("starting server ", "host", host, "port", port)
	srv, err := wish.NewServer(

		wish.WithAddress(net.JoinHostPort(host, port)),

		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithKeyboardInteractiveAuth(func(ctx ssh.Context, challenger gossh.KeyboardInteractiveChallenge) bool {
			log.Info("keyboard interactive challenge")
			answers, err := challenger(
				"", `Possible answers are "vim" or "other"`, []string{"What is the best ide?"}, []bool{true},
			)
			if err != nil {
				log.Error("Error with answers", "error", err)
				return false
			}
			return len(answers) == 1 && answers[0] == "vim"
		}),

		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Info("Starting SSH server", "host", host, "port", port)
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer func() { cancel() }()
	log.Info("Stopping SSH server")
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	contentHeight := pty.Window.Height - headerHeight - footerHeight
	renderer := bubbletea.MakeRenderer(s)
	txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))
	quitStyle := renderer.NewStyle().Foreground(lipgloss.Color("15"))
	headerStyle := renderer.NewStyle().Bold(true).Background(lipgloss.Color("62")).PaddingLeft(2)
	projectsPosts, err := loadProjects()
	if err != nil {
		log.Error("Failed to load projects", "error", err)
		// Fall back to empty projects list
		projectsPosts = []Projects{}
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
	m := model{
		term:           pty.Term,
		profile:        renderer.ColorProfile().Name(),
		width:          pty.Window.Width,
		height:         pty.Window.Height,
		bg:             bg,
		txtStyle:       txtStyle,
		quitStyle:      quitStyle,
		headerStyle:    headerStyle,
		viewport:       vp,
		content:        "",
		projectsPosts:  projectsPosts,
		inProjectsList: true,
		projectsList:   projectsList,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type model struct {
	term           string
	state          string
	profile        string
	width          int
	height         int
	bg             string
	txtStyle       lipgloss.Style
	quitStyle      lipgloss.Style
	headerStyle    lipgloss.Style
	viewport       viewport.Model
	content        string
	projectsPosts  []Projects
	selectedPost   *Projects
	inProjectsList bool
	projectsList   list.Model
}

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Background(lipgloss.Color("#25A065")).Padding(0, 1)
	listStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#874BFD")).Padding(0, 0)
)

type ProjectsFile struct {
	Projects []Projects `json:"projects"`
}

func loadProjects() ([]Projects, error) {
	data, err := os.ReadFile("projects.txt")
	if err != nil {
		return nil, err
	}

	// Split the file by project separator
	projectTexts := strings.Split(string(data), "---")
	var projects []Projects

	for _, text := range projectTexts {
		if strings.TrimSpace(text) == "" {
			continue
		}

		// Parse each project
		lines := strings.Split(strings.TrimSpace(text), "\n")
		var project Projects
		var contentLines []string

		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Title:") {
				project.ProjectTitle = strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
			} else if strings.HasPrefix(line, "Number:") {
				num, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Number:")))
				project.ProjectNumber = num
			} else if line != "" || i > 2 { // After title and number, collect content
				contentLines = append(contentLines, line)
			}
		}

		project.ProjectContent = strings.TrimSpace(strings.Join(contentLines, "\n"))
		if project.ProjectTitle != "" { // Only add if we have at least a title
			projects = append(projects, project)
		}
	}

	return projects, nil
}

type Projects struct {
	ProjectTitle   string `json:"title"`
	ProjectContent string `json:"content"`
	ProjectNumber  int    `json:"number"`
}

func (p Projects) Title() string { return fmt.Sprintf("%d. %s", p.ProjectNumber, p.ProjectTitle) }
func (p Projects) Description() string {
	if len(p.ProjectContent) > 100 {
		return p.ProjectContent[:100] + "..."
	}
	return p.ProjectContent
}
func (p Projects) FilterValue() string { return p.ProjectTitle }
func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight
		m.projectsList.SetWidth(msg.Width)
		m.projectsList.SetHeight(msg.Height - headerHeight - footerHeight - 2)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.viewport.LineDown(1)
		case "k", "up":
			m.viewport.LineUp(1)
		case "d":
			m.viewport.LineDown(10)
		case "u":
			m.viewport.LineUp(10)
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		case "o":
			m.state = "home"
		case "backspace":
			if m.state == "projects" && !m.inProjectsList {
				m.inProjectsList = true
				m.selectedPost = nil
			}
		case "p":
			m.state = "projects"
			m.inProjectsList = true
		case "r":
			m.state = "resume"
			m.viewport.SetContent(getResumeContent())
		case "c":
			m.state = "contact"
			m.viewport.SetContent(getContactContent())
		case "enter":
			if m.state == "projects" && m.inProjectsList {
				if i, ok := m.projectsList.SelectedItem().(Projects); ok {
					m.selectedPost = &i
					m.inProjectsList = false
					m.viewport.SetContent(i.ProjectContent)
					m.viewport.GotoTop()
				}
			}
		default:
			if m.state == "projects" && m.inProjectsList {
				if num, err := strconv.Atoi(msg.String()); err == nil && num >= 0 && num < len(m.projectsPosts) {
					m.selectedPost = &m.projectsPosts[num]
					m.inProjectsList = false
					m.viewport.SetContent(m.selectedPost.ProjectContent)
					m.viewport.GotoTop()

				}
			}

		}
	}
	if m.state == "projects" {
		if m.inProjectsList {
			var cmd tea.Cmd
			m.projectsList, cmd = m.projectsList.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func getResumeContent() string {
	return `
EDUCATION
University Of ******** (2027)
BSc Computer Science with Artificial Intelligence- *******, UK

WORK EXPERIENCE
********** Systems Engineer intern (2025)
• Assembled PCB's via reflow oven etc
• Start to finish designed and modelled *********

***** ****** Software developer (2025) 
• Created entire stack, backend to front-end 
• Project was sensor based 

Software Development Internship (2024)
***** and **** - ******** , UK
• Core Rust developer for CO2 recording software
• Implemented cross-platform release workflow
• Designed and created HTTP server/DB infrastructure

TECHNICAL SKILLS
Primary:
• GoLang, Rust, Devops, Linux
Secondary:
• STM32CubeIDE with C, Python, Java, React, NodeJS, Arduino C++/ESP-IDF

INTERESTS
Hackathons, Weightlifting, Running, 3D Printing, Electronics
`
}

func getContactContent() string {
	return `
Email: w@willx86.com
Github: github.com/will-x86
    `
}

func (m model) View() string {
	header := m.headerStyle.Width(m.width).Render("willx86.com")

	contentHeight := m.height - headerHeight - footerHeight

	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight)

	var content string
	switch m.state {
	case "home":
		content = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render(homeText)
	case "projects":
		if m.inProjectsList {
			content = contentStyle.
				Render(m.projectsList.View())
		} else if m.selectedPost != nil {
			content = contentStyle.
				Render(m.viewport.View())
		}
	case "resume":
		content = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render(getResumeContent())
	case "contact":
		content = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render(getContactContent())
	default:
		content = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render("Welcome! Use the controls below to navigate.")
	}

	controls := m.quitStyle.Render("q: quit • o: home • p: projects • r: resume • c: contact")
	if m.state == "projects" && m.inProjectsList {
		controls += m.quitStyle.Render(" • [0-9]: select post")
	}
	if m.state == "projects" && !m.inProjectsList {
		controls += m.quitStyle.Render(" • backspace: back to posts • j/k | d/u | up/down to scroll")
	}

	footer := lipgloss.NewStyle().
		Width(m.width).
		Height(footerHeight).
		AlignVertical(lipgloss.Bottom).
		Render(controls)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Height(headerHeight).Render(header),
		content,
		footer,
	)
}
