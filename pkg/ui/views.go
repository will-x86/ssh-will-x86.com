package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

const homeText = `
Intro:
Hi, I'm will-x86, and this is my personal website (sshite?).
I've been developing software since early 2018 & mainly work with Go & Rust.
More recently I've been open to other technologies, 
primarily micro-electronics and front-end (Next/React).

About myself:
- I self-host, from Ollama to Immich I love it all
- I'm a University student in the UK
- I'm into it all, 3D printing to serverless to e-ink readers
- I'm starting to love designing PCB's....
`

func (m Model) View() string {
	header := m.HeaderStyle.Width(m.width).Render("willx86.com")

	contentHeight := m.height - HeaderHeight - FooterHeight
	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight)

	var body string
	switch m.State {
	case StateHome:
		body = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render(homeText)
	case StateProjects:
		if m.inProjectsList {
			body = contentStyle.Render(m.projectsList.View())
		} else if m.selectedPost != nil {
			body = contentStyle.Render(m.viewport.View())
		}
	case StateContact:
		body = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render(contactContent())
	case StateBlog:
		body = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render(blogContent())
	case StateMessages:
		body = contentStyle.
			Align(lipgloss.Center, lipgloss.Top).
			Render(m.messagesContent())
	default:
		body = contentStyle.
			Align(lipgloss.Center, lipgloss.Center).
			Render("Welcome! Use the controls below to navigate.")
	}

	controls := m.QuitStyle.Render("q: quit • o: home • p: projects • b: blog • c: contact • m: message me!")
	if m.State == StateProjects && m.inProjectsList {
		controls += m.QuitStyle.Render(" • [0-9]: select post")
	}
	if m.State == StateProjects && !m.inProjectsList {
		controls += m.QuitStyle.Render(" • backspace: back to posts • j/k | d/u | up/down to scroll")
	}

	footer := lipgloss.NewStyle().
		Width(m.width).
		Height(FooterHeight).
		AlignVertical(lipgloss.Bottom).
		Render(controls)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Height(HeaderHeight).Render(header),
		body,
		footer,
	)
}

func blogContent() string {
	return `See w.willx86.com
	Mostly mundane small tutorials, maybe I'll do something more with it one day...
	Update! You can now see how I made the "message" feature you can see by pressing 'm'`
}

func contactContent() string {
	return `
Email: w@willx86.com
Github: github.com/will-x86
    `
}

func (m Model) messagesContent() string {
	if m.messageSent {
		return `
Thank you for your message!

It's currently making it's way through the internet.
After that it'll be permanently burned into thermal receipt paper, on my desk

See https://w.willx86.com/2025/11/06/printing-messages-from-my-site.html for more details ! 


Press 'o' to return home or 'm' to send another message.
`
	}
	if m.tooLong {
		return `
		Please try again! Your message was too long and or had too many newlines :) paper's expensive yaknow?
		`
	}
	if m.editingName {
		return fmt.Sprintf(`
Leave a message 

Change your name:
%s

Press Enter to confirm | Esc to cancel
`, m.nameInput.View())
	}
	return fmt.Sprintf(`
Leave a message 

Signed in as: %s

%s

Press Ctrl+N to change name | Ctrl+S to send | Esc to cancel
`, m.username, m.messageInput.View())
}
