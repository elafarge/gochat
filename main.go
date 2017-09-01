package main

// TODO handle errors on websocket writes

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

var (
	// CloseErrors is the list of websocket errors we close the connection for
	CloseErrors = []int{
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseProtocolError,
		websocket.CloseUnsupportedData,
		websocket.CloseNoStatusReceived,
		websocket.CloseAbnormalClosure,
		websocket.CloseInvalidFramePayloadData,
		websocket.ClosePolicyViolation,
		websocket.CloseMessageTooBig,
		websocket.CloseMandatoryExtension,
		websocket.CloseInternalServerErr,
		websocket.CloseServiceRestart,
		websocket.CloseTryAgainLater,
		websocket.CloseTLSHandshake,
	}

	// Upgrader configuration
	Upgrader = websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

// Message describes the structure of our "message" payloads
type Message struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Content   string `json:"content"`
}

// WebSocketHandler manages bidirectional websocket streams between senders and recipients
type WebSocketHandler struct {
	Conf *ChatRelayConf

	broker Broker
}

func (h WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithField("ctx", "HTTPHandler").
			Errorf("Error upgrading connection to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	consumer := r.FormValue("user")
	if len(consumer) == 0 {
		conn.WriteMessage(
			websocket.ClosePolicyViolation,
			[]byte(`{"message": "Your request should contain a "user" parameter"}`),
		)
		return
	}

	// Let's create our Websocket write mutex and wrap in into a send
	var mut sync.Mutex

	sendWSMessage := func(message []byte) {
		mut.Lock()
		err := conn.WriteMessage(websocket.TextMessage, message)
		mut.Unlock()
		if err != nil {
			log.WithFields(log.Fields{
				"ctx":  "WebSocketHandler",
				"user": consumer,
			}).Errorf("Error sending message to browser: %v", err)
		}
	}

	// Let's listen for incoming messages from peers in a separate goroutine
	go func() {
		msgChan, err := h.broker.Subscribe(consumer)
		if err != nil {
			// It's always good to add some contexti (tags) to our logs. Would we push them into a log
			// parsing solution such as logmatic, splunk, or a Elasticsearch-Fluentd-Kibana stack, this
			// would enable us to see all logs for a given consumer for instance.
			log.WithFields(log.Fields{
				"ctx":  "WebSocketHandler",
				"user": consumer,
			}).Errorf("Error listening to incoming messages: %v", err)
		}

		for {
			incoming := <-msgChan
			sendWSMessage(incoming)
		}
	}()

	// Let's listen for new message in the goroutine processing the request
	for {
		// Note: we're not using conn.ReadMessage() here since it calls ioutil.ReadAll() and makes DDoS
		// attacks easy: sending an infinite message will overload the RAM.
		_, reader, err := conn.NextReader()
		if err != nil {
			if !websocket.IsCloseError(err, CloseErrors...) {
				log.WithFields(log.Fields{
					"ctx":  "WebSocketHandler",
					"user": consumer,
				}).Errorf("Error getting next message reader: %v", err)
				continue
			}
			log.WithFields(log.Fields{
				"ctx":  "WebSocketHandler",
				"user": consumer,
			}).Debugf("Close error received: %v", err)
			break
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
			log.WithFields(log.Fields{
				"ctx":  "WebSocketHandler",
				"user": consumer,
			}).Debugf("Message is too big, dropping it")
			sendWSMessage([]byte(`{"message": "too_big"}`))
			continue
		}

		if readErr != nil {
			log.WithFields(log.Fields{
				"ctx":  "WebSocketHandler",
				"user": consumer,
			}).Debugf("Error reading message: %v", readErr)

			sendWSMessage([]byte(fmt.Sprintf("{\"message\": \"read_error\": \"%v\"}", readErr)))
			continue
		}

		log.WithFields(log.Fields{
			"ctx":  "WebSocketHandler",
			"user": consumer,
		}).Debugf("Received message %s", string(data))
		var message Message
		// This Unmarshal call could actually be avoided if we used a "recipient" query parameter.
		// However, I'm leaving this since, at least to perform some validation, one would probably like
		// to unmarshal the Payload
		err = json.Unmarshal(data, &message)
		if err != nil {
			log.WithFields(log.Fields{
				"ctx":  "WebSocketHandler",
				"user": consumer,
			}).Errorf("Error unmarshaling message to JSON: %v", err)
		}

		if !h.broker.HasTopic(message.Recipient) {
			sendWSMessage([]byte(`{"message": "not_found"}`))
			continue
		}

		err = h.broker.Publish(message.Recipient, data)
		if err != nil {
			sendWSMessage([]byte(fmt.Sprintf(
				"{\"message\": \"Error sending message to %s: %v\"}", message.Recipient, err,
			)))
		}
		sendWSMessage([]byte(`{"message": "ok"}`))
	}

	h.broker.Unsubscribe(consumer)
	log.WithFields(log.Fields{
		"ctx":  "HTTPHandler",
		"user": consumer,
	}).Debugf("Connection terminated with success")
}

func main() {
	conf, err := NewChatRelayConfFromFlags()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	handler := WebSocketHandler{
		Conf: conf,

		broker: NewInMemoryBroker(),
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
