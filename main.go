package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/elapsed"
	"github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"
)

const (
	host = "localhost"
	port = "23234"
)

//go:embed banner.txt
var banner string

func main() {
	srv, err := wish.NewServer(

		wish.WithAddress(net.JoinHostPort(host, port)),

		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithBannerHandler(func(ctx ssh.Context) string {
			return fmt.Sprintf(banner, ctx.User())
		}),
		wish.WithKeyboardInteractiveAuth(func(ctx ssh.Context, challenger gossh.KeyboardInteractiveChallenge) bool {
			log.Info("keyboard interactive challenge")
			answers, err := challenger(
				"", `Possible answers are "vim" or "other"`, []string{"Best ide?"}, []bool{true},
			)
			if err != nil {
				log.Error("Error with answers", "error", err)
				return false
			}
			return len(answers) == 1 && answers[0] == "vim"
		}),

		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					wish.Println(sess, fmt.Sprintf("Hello %s", sess.User()))
					next(sess)
				}
			},

			// The last item in the chain is the first to be called.
			logging.Middleware(),
			elapsed.Middleware(),
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
