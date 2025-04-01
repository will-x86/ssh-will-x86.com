package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	lorem "github.com/derektata/lorem/ipsum"

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
	host         = "localhost"
	port         = "23234"
	headerHeight = 1
	footerHeight = 1
)

func main() {
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
	blogPosts := []BlogPost{
		{
			title:   "first",
			content: getBlogContent(),
			number:  0,
		},
		{
			title:   "2nd",
			content: getBlogContent(),
			number:  1,
		},
		{
			title:   "3ds",
			content: getBlogContent(),
			number:  2,
		},
	}
	items := make([]list.Item, len(blogPosts))
	for i, post := range blogPosts {
		items[i] = post
	}
	delegate := list.NewDefaultDelegate()
	blogList := list.New(items, delegate, pty.Window.Width, contentHeight-2)
	blogList.SetShowHelp(false)
	blogList.SetShowTitle(false)
	blogList.SetFilteringEnabled(false)
	blogList.Styles.PaginationStyle = lipgloss.NewStyle()

	bg := "light"
	if renderer.HasDarkBackground() {
		bg = "dark"
	}
	vp := viewport.New(pty.Window.Width, contentHeight)
	vp.Style = renderer.NewStyle().Border(lipgloss.RoundedBorder())
	m := model{
		term:        pty.Term,
		profile:     renderer.ColorProfile().Name(),
		width:       pty.Window.Width,
		height:      pty.Window.Height,
		bg:          bg,
		txtStyle:    txtStyle,
		quitStyle:   quitStyle,
		headerStyle: headerStyle,
		viewport:    vp,
		content:     getBlogContent(),
		blogPosts:   blogPosts,
		inBlogList:  true,
		blogList:    blogList,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func getBlogContent() string {
	g := lorem.NewGenerator()
	g.SentencesPerParagraph = 5
	var b string
	for range 100 {
		b += g.Generate(5) + "\n"
	}
	return b
}

type model struct {
	term         string
	state        string
	profile      string
	width        int
	height       int
	bg           string
	txtStyle     lipgloss.Style
	quitStyle    lipgloss.Style
	headerStyle  lipgloss.Style
	viewport     viewport.Model
	content      string
	blogPosts    []BlogPost
	selectedPost *BlogPost
	inBlogList   bool
	blogList     list.Model
}

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Background(lipgloss.Color("#25A065")).Padding(0, 1)
	listStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#874BFD")).Padding(0, 0)
)

type BlogPost struct {
	title   string
	content string
	number  int
}

func (b BlogPost) Title() string       { return fmt.Sprintf("%d. %s", b.number, b.title) }
func (b BlogPost) Description() string { return b.content[:100] + "..." }
func (b BlogPost) FilterValue() string { return b.title }
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
		m.blogList.SetWidth(msg.Width)
		m.blogList.SetHeight(msg.Height - headerHeight - footerHeight - 2)
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
			if m.state == "blog" && !m.inBlogList {
				m.inBlogList = true
				m.selectedPost = nil
			}
		case "b":
			m.state = "blog"
			m.viewport.SetContent(getBlogContent())
		case "r":
			m.state = "resume"
			m.viewport.SetContent(getResumeContent())
		case "c":
			m.state = "contact"
			m.viewport.SetContent(getContactContent())
		case "enter":
			if m.state == "blog" && m.inBlogList {
				if i, ok := m.blogList.SelectedItem().(BlogPost); ok {
					m.selectedPost = &i
					m.inBlogList = false
					m.viewport.SetContent(i.content)
					m.viewport.GotoTop()

				}
			}
		default:
			if m.state == "blog" && m.inBlogList {
				if num, err := strconv.Atoi(msg.String()); err == nil && num >= 0 && num < len(m.blogPosts) {
					m.selectedPost = &m.blogPosts[num]
					m.inBlogList = false
					m.viewport.SetContent(m.selectedPost.content)
					m.viewport.GotoTop()

				}
			}

		}
	}
	if m.state == "blog" {
		if m.inBlogList {
			var cmd tea.Cmd
			m.blogList, cmd = m.blogList.Update(msg)
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
	return "resume"
}

func getContactContent() string {
	return "contact idk@bob.com"
}

func (m model) View() string {
	header := m.headerStyle.Width(m.width).Render("will-x86.com")

	contentHeight := m.height - headerHeight - footerHeight

	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight)

	var content string
	switch m.state {
	case "home":
		content = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render("Welcome to my personal website!")
	case "blog":
		if m.inBlogList {
			content = contentStyle.
				Render(m.blogList.View())
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

	controls := m.quitStyle.Render("q: quit • o: home • b: blog • r: resume • c: contact")
	if m.state == "blog" && m.inBlogList {
		controls += m.quitStyle.Render(" • [0-9]: select post")
	}
	if m.state == "blog" && !m.inBlogList {
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
