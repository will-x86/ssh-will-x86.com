package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/will-x86/ssh-will-x86/pkg/content"
	"github.com/will-x86/ssh-will-x86/pkg/server"
)

func countRune(s string, r rune) int {
	count := 0
	for _, c := range s {
		if c == r {
			count++
		}
	}
	return count
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - HeaderHeight - FooterHeight
		m.projectsList.SetWidth(msg.Width)
		m.projectsList.SetHeight(msg.Height - HeaderHeight - FooterHeight - 2)
		m.messageInput.SetWidth(msg.Width - 4)

	case tea.KeyMsg:
		// Messages state gets its own key handling before the global switch.
		if m.State == StateMessages && !m.messageSent {
			return m.updateMessages(msg)
		}

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
			m.State = StateHome
		case "backspace":
			if m.State == StateProjects && !m.inProjectsList {
				m.inProjectsList = true
				m.selectedPost = nil
			}
		case "b":
			m.State = StateBlog
			m.viewport.SetContent(blogContent())
		case "p":
			m.State = StateProjects
			m.inProjectsList = true
		case "c":
			m.State = StateContact
			m.viewport.SetContent(contactContent())
		case "m":
			m.State = StateMessages
			m.messageSent = false
			m.editingName = false
			m.tooLong = false
			m.messageInput.Focus()
		case "enter":
			if m.State == StateProjects && m.inProjectsList {
				if i, ok := m.projectsList.SelectedItem().(content.Project); ok {
					m.selectedPost = &i
					m.inProjectsList = false
					m.viewport.SetContent(i.ProjectContent)
					m.viewport.GotoTop()
				}
			}
		default:
			if m.State == StateProjects && m.inProjectsList {
				if num, err := strconv.Atoi(msg.String()); err == nil && num >= 0 && num < len(m.projectsPosts) {
					m.selectedPost = &m.projectsPosts[num]
					m.inProjectsList = false
					m.viewport.SetContent(m.selectedPost.ProjectContent)
					m.viewport.GotoTop()
				}
			}
		}
	}

	// Delegate list / viewport updates when in projects.
	if m.State == StateProjects {
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

// Key events for message state only
func (m Model) updateMessages(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editingName {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter", "esc":
			if strings.TrimSpace(m.nameInput.Value()) != "" {
				m.username = strings.TrimSpace(m.nameInput.Value())
			}
			m.editingName = false
			m.nameInput.Blur()
			m.messageInput.Focus()
			return m, nil
		default:
			var cmd tea.Cmd
			m.nameInput, cmd = m.nameInput.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.State = StateHome
		m.messageInput.Reset()
		return m, nil
	case "ctrl+n":
		m.editingName = true
		m.nameInput.SetValue(m.username)
		m.nameInput.Focus()
		m.messageInput.Blur()
		return m, textinput.Blink
	case "ctrl+s":
		content := strings.TrimSpace(m.messageInput.Value())
		if content != "" {
			if countRune(content, '\n') >= 10 || len(content) >= 1000 {
				log.Infof("Message too long: %s", content)
				m.tooLong = true
				m.messageInput.Reset()
			} else {
				server.AddMessage(m.username, content)
				m.messageSent = true
				m.tooLong = false
				m.messageInput.Reset()
			}
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.messageInput, cmd = m.messageInput.Update(msg)
		return m, cmd
	}
}
