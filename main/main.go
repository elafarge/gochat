package main

// TODO handle errors on websocket writes

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/elafarge/gochat"
	"github.com/elafarge/gochat/broker"
)

// Upgrader configuration
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WebSocketHandler manages bidirectional websocket streams between senders and recipients
type WebSocketHandler struct {
	Conf *gochat.Conf

	broker broker.Broker
}

// ServeHTTP is our HTTP Handler: it essentially handles Websocket connection
// upgrade. The logic behind the Websocket connection itself is wrapped into
// the ChatRelay struct.
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

	relay := &gochat.Relay{
		Conf:         h.Conf,
		ConsumerName: consumer,
		Broker:       h.broker,
		Conn:         conn,
	}

	// Launch a separate goroutine that forwards messages from the broker to the client
	go relay.Receive()

	// Forward message from the client to the broker
	relay.Send()

	log.WithFields(log.Fields{
		"ctx":  "HTTPHandler",
		"user": consumer,
	}).Debugf("Connection terminated with success")
}

func main() {
	conf, err := gochat.NewConfFromFlags()
	if err != nil {
		log.WithField("ctx", "Main").Fatalf("Configuration error: %v", err)
	}

	handler := WebSocketHandler{
		Conf: conf,

		broker: broker.NewInMemoryBroker(),
	}

	log.WithField("ctx", "Main").Infoln("Starting gochat websocket server on %s", conf.ListenOn)
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
		log.WithField("ctx", "Main").Errorf("Error in server loop: %v", err)
	}
}
