package main

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
)

func WebServer(port string) {
	http.HandleFunc("/messages/latest", recoverWrap(handler))

	log.Infof("Starting webserver on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Errorf("Server stopped: %v", err)
		WebServer(port)
	}
}
func recoverWrap(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Errorf("Recovered from panic: %v", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		h(w, r)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	auth := r.URL.Query().Get("secret")
	log.Infof("Got auth: %s", auth)

	if auth != *secretKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Info("Hit messages ep")
	msgs := getMessages()
	if len(msgs) != 0 {
		first := msgs[0]
		log.Infof("Printing message %s", first.Content)
		removeMessage(first.From, first.Content)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprintf(w, "%s---%s---%s", first.From, first.Content, first.Timestamp)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
