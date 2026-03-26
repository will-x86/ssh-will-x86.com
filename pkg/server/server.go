package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

var secretKey string

func WebServer(port, sk string) {
	secretKey = sk
	http.HandleFunc("/messages/latest", recoverWrap(handler))

	log.Infof("Starting webserver on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Errorf("Server stopped: %v", err)
		WebServer(port, secretKey)
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

func removeMessage(from, content string) {
	messagesMu.Lock()
	defer messagesMu.Unlock()

	for i := range messages {
		if messages[i].Content == content && messages[i].From == from {
			messages = append(messages[:i], messages[i+1:]...)
			break
		}
	}
}

type Message struct {
	From      string
	Content   string
	Timestamp time.Time
}

var (
	messages   []Message
	messagesMu sync.RWMutex
)

func AddMessage(from, content string) {
	messagesMu.Lock()
	defer messagesMu.Unlock()
	messages = append(messages, Message{
		From:      from,
		Content:   content,
		Timestamp: time.Now(),
	})
	log.Info("New message saved", "from", from, "content", content)
}
func getMessages() []Message {
	messagesMu.RLock()
	defer messagesMu.RUnlock()
	msgCopy := make([]Message, len(messages))
	copy(msgCopy, messages)
	return msgCopy
}
func handler(w http.ResponseWriter, r *http.Request) {
	auth := r.URL.Query().Get("secret")
	log.Infof("Got auth: %s", auth)

	if auth != secretKey {
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
