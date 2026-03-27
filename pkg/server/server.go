package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

var secretKey string
var workerURL string
var workerSecret string

func WebServer(port, sk, wURL, wSecret string) {
	secretKey = sk
	workerURL = wURL
	workerSecret = wSecret
	http.HandleFunc("/messages/latest", recoverWrap(handler))

	log.Infof("Starting webserver on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Errorf("Server stopped: %v", err)
		WebServer(port, secretKey, workerURL, workerSecret)
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
	From      string    `json:"from"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	messages   []Message
	messagesMu sync.RWMutex
)

func AddMessage(from, content string) {
	ts := time.Now()

	messagesMu.Lock()
	messages = append(messages, Message{
		From:      from,
		Content:   content,
		Timestamp: ts,
	})
	messagesMu.Unlock()

	log.Info("New message saved", "from", from, "content", content)

	if workerURL != "" {
		go func() {
			body, err := json.Marshal(map[string]string{
				"from":      from,
				"content":   content,
				"timestamp": ts.Format(time.RFC3339Nano),
			})
			if err != nil {
				log.Errorf("Worker: failed to marshal message: %v", err)
				return
			}
			url := workerURL + "/message?secret=" + workerSecret
			resp, err := http.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				log.Errorf("Worker: POST /message failed: %v", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Errorf("Worker: POST /message returned %d", resp.StatusCode)
			}
		}()
	}
}

func getMessages() []Message {
	messagesMu.RLock()
	defer messagesMu.RUnlock()
	msgCopy := make([]Message, len(messages))
	copy(msgCopy, messages)
	return msgCopy
}

// fetchFromWorker GET /next on the Worker and returns the raw plain-text
// response body. Returns ("", false) if the queue is empty, ("", true) on error.
func fetchFromWorker() (string, bool) {
	url := workerURL + "/next?secret=" + workerSecret
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		log.Errorf("Worker: GET /next failed: %v", err)
		return "", true
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return "", false
	}
	if resp.StatusCode != http.StatusOK {
		log.Errorf("Worker: GET /next returned %d", resp.StatusCode)
		return "", true
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Worker: reading /next body failed: %v", err)
		return "", true
	}
	return string(data), false
}

func handler(w http.ResponseWriter, r *http.Request) {
	auth := r.URL.Query().Get("secret")

	if auth != secretKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// If a Worker is configured, use it
	if workerURL != "" {
		body, workerErr := fetchFromWorker()
		if !workerErr {
			if body == "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			log.Infof("Got message from worker: %s", body)
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, body)
			return
		}
		log.Warn("Worker unavailable, falling back to in-memory store")
	}

	// In-memory fallback
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
