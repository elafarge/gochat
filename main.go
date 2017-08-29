package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Upgrader configuration
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Message describes the structure of our "message" payloads
type Message struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

// WebSocketHandler manages bidirectional websocket streams between senders and recipients
type WebSocketHandler struct {
	Conf *ChatRelayConf

	writeMutex sync.Mutex
}

func (h WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithField("ctx", "HTTPHandler").
			Errorln("Error upgrading connection to WebSocket: %v", err)
		return
	}

	for {
		// Note: we're not using conn.ReadMessage() here since it calls ioutil.ReadAll() and makes DDoS
		// attacks easy: sending an infinite message will overload the RAM.
		_, reader, err := conn.NextReader()
		if err != nil {
			log.WithField("ctx", "ReadMessage").Errorf("Error getting next message reader: %v", err)
			continue
		}

		// Read until EOF or abort if buffer is full
		data := make([]byte, h.Conf.MaxMsgSize)
		offset := 0
		tooBig := false
		var readErr error
		for {
			if offset >= h.Conf.MaxMsgSize {
				tooBig = true
				break
			}
			n, err := reader.Read(data[offset:])
			offset += n

			if err == io.EOF {
				data = data[:offset]
				break
			}

			if err != nil {
				readErr = err
				break
			}
		}
		if tooBig {
			log.WithField("ctx", "ReadMessage").Debugf("Message is too big, dropping it")
			h.writeMutex.Lock()
			conn.WriteMessage(websocket.TextMessage, []byte(`{"message": "too_big"}`))
			h.writeMutex.Unlock()
			continue
		}

		if readErr != nil {
			log.WithField("ctx", "ReadMessage").Debugf("Error reading message: %v", readErr)
			h.writeMutex.Lock()
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("{\"message\": \"read_error\": \"%v\"}", readErr)))
			h.writeMutex.Unlock()
			continue
		}

		log.WithField("ctx", "ReadMessage").Debugf("Received message %s", string(data))
		var message Message
		err = json.Unmarshal(data, &message)
		if err != nil {
			log.WithField("ctx", "ReadMessage").Errorf("Error unmarshaling message to JSON: %v", err)
		}

		// if b.HasTopic(message.Recipient) {
		// 	b.Send(data)
		// } else {
		// 	conn.WriteMessage(websocket.TextMessage, []byte(`{"message": "not_found"}`))
		// }
		h.writeMutex.Lock()
		conn.WriteMessage(websocket.TextMessage, []byte(`{"message": "ok"}`))
		h.writeMutex.Unlock()

	}

	w.WriteHeader(200)
	w.Write([]byte(`{"message": "Connection terminated with success"}`))
}

func main() {
	conf, err := NewChatRelayConfFromFlags()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	handler := WebSocketHandler{
		Conf: conf,
	}

	log.WithField("ctx", "server-main").Infoln("Starting gochat websocket server")
	s := http.Server{
		Addr:    conf.ListenOn,
		Handler: handler,

		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       1 * time.Minute,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}
	defer s.Close()

	err = s.ListenAndServe()
	if err != nil {
		log.WithField("ctx", "server-main").Errorf("Error in server loop: %v", err)
	}
}
