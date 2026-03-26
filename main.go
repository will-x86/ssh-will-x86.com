package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/will-x86/ssh-will-x86/pkg/server"
	sshserver "github.com/will-x86/ssh-will-x86/pkg/ssh"
	"github.com/will-x86/ssh-will-x86/pkg/ui"
)

var (
	hostFlag      = flag.String("host", "0.0.0.0", "Host to listen on")
	portFlag      = flag.String("port", "22", "Port to listen on")
	webServerPort = flag.String("webserver-port", "9000", "Port for the HTTP message server")
	secretKey     = flag.String("sK", os.Getenv("SECRET_KEY"), "Secret key for the message endpoint")
)

func main() {
	flag.Parse()
	if *secretKey == "" {
		panic("no key set")
	}

	go server.WebServer(*webServerPort, *secretKey)

	srv, err := sshserver.NewServer(*hostFlag, *portFlag, ui.NewTeaHandler())
	if err != nil {
		log.Error("Could not create SSH server", "error", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("Starting SSH server", "host", *hostFlag, "port", *portFlag)
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	log.Info("Stopping SSH server")
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}
