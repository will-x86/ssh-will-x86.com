package ssh

import (
	"net"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"
)

func NewServer(host, port string, handler bubbletea.Handler) (*ssh.Server, error) {
	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithKeyboardInteractiveAuth(authChallenge),
		wish.WithMiddleware(
			bubbletea.Middleware(handler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, err
	}
	return srv, nil
}

// vim questions
func authChallenge(ctx ssh.Context, challenger gossh.KeyboardInteractiveChallenge) bool {
	log.Info("keyboard interactive challenge")
	answers, err := challenger(
		"", `Possible answers are "vim" or "other"`,
		[]string{"What is the best ide?"},
		[]bool{true},
	)
	if err != nil {
		log.Error("Error with answers", "error", err)
		return false
	}
	return len(answers) == 1 && answers[0] == "vim"
}
