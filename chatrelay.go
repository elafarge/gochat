package gochat

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/elafarge/gochat/broker"
)

// CloseErrors is the list of websocket errors we close the connection for
var CloseErrors = []int{
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

// Message describes the structure of our "message" payloads
type Message struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Content   string `json:"content"`
}

// Relay handles a connection with a given client: it consumes messages targeting this client
// from a broker and forwards them while registering sent messages against the broker
type Relay struct {
	ID     string
	Conf   *Conf
	Topic  string
	Broker broker.Broker
	Conn   *websocket.Conn

	// A mutex to ensure only one goroutine writes on our websocket at the time
	mut sync.Mutex
}

// Log returns a logger instance with Relay related context
func (r *Relay) Log(ctx string) *log.Entry {
	return log.WithFields(log.Fields{
		"component": "Relay",
		"ctx":       ctx,
		"user":      r.Topic,
	})
}

// SendToBrowser is used to send message to the client
func (r *Relay) SendToBrowser(message []byte) error {
	r.mut.Lock()
	defer r.mut.Unlock()
	return r.Conn.WriteMessage(websocket.TextMessage, message)
}

// Receive consumes incoming messages from the broker and forwards them
// to the client through the websocket connection
func (r *Relay) Receive() {
	msgChan, err := r.Broker.Subscribe(r.Topic, r.ID)
	if err != nil {
		// It's always good to add some contextxi (tags) to our logs. Would we push them into a log
		// parsing solution such as logmatic, splunk, or a Elasticsearch-Fluentd-Kibana stack, this
		// would enable us to see all logs for a given consumer for instance.
		errMsg := fmt.Sprintf("Error listening to incoming messages: %v", err)
		r.Log("Receive").Errorf(errMsg)
		r.SendToBrowser([]byte(fmt.Sprintf("{\"error\": \"%s\"}", errMsg)))
	}

	// ranging over a channel allows us to have this function exit (and associated goroutine stop)
	// when close() is called on the channel msgChan
	for incoming := range msgChan {
		if err := r.SendToBrowser(incoming); err != nil {
			r.Log("Receive").Infof("Error sending message to browser: %v", err)
		}
	}
}

// Send reads messages from the Websocket connection and sends them to the broker
// NOTE: send consumes from the Websocket reader in a RAM-savy way, dropping payloads that are over
// a defined threshold
func (r *Relay) Send() {
	for {
		// Note: we're not using conn.ReadMessage() here since it calls ioutil.ReadAll() and makes DDoS
		// attacks easy: sending an infinite message will overload the RAM.

		// Let's get the next reader and handle connection errors accordingly
		_, reader, err := r.Conn.NextReader()
		if err != nil {
			break
		}

		// Read until EOF or abort if buffer is full
		data, err := r.ReadWSMessage(reader)
		if err != nil {
			// "Too big" error messages shouldn't flood stdout (and any log collecting pipeline after
			// that) when people are trying to DDoS our service
			r.Log("Send").Debug(err)
			r.SendToBrowser([]byte(fmt.Sprintf("{\"error\": \"%v\"}", err)))
			break
		}

		r.Log("Send").Debugf("Received message %s", string(data))
		var message Message

		err = json.Unmarshal(data, &message)
		if err != nil {
			r.sendErr("Error unmarshaling message to JSON: %v", err)
			continue
		}

		err = r.Broker.Publish(message.Recipient, data)
		if err != nil {
			r.sendErr("Error sending message to %s: %v", message.Recipient, err)
		}
		r.SendToBrowser([]byte(`{"ok": "message sent with success"}`))
	}

	// Let's unsubscribe from the broker so that it can delete resources allocated to our send/receive
	// goroutines if needs be
	err := r.Broker.Unsubscribe(r.Topic, r.ID)
	if err != nil {
		r.Log("Send").Errorf("Error unsubscribing from broker: %v", err)
	}
}

// ReadWSMessage reads a websocket message while making sure we don't load more
// than a certain threshold in RAM
func (r *Relay) ReadWSMessage(reader io.Reader) (data []byte, err error) {

	data = make([]byte, r.Conf.MaxMsgSize)
	offset := 0
	tooBig := false
	var readErr error
	for {
		if offset >= r.Conf.MaxMsgSize {
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
		return nil, fmt.Errorf("Message is too big, dropping it")
	}

	if readErr != nil {
		return nil, fmt.Errorf("Error reading message: %v", readErr)
	}

	return data, nil
}

func (r *Relay) sendErr(msg string, vars ...interface{}) {
	if len(vars) > 0 {
		msg = fmt.Sprintf(msg, vars)
	}
	r.Log("Send").Debugf(msg)
	r.SendToBrowser([]byte(fmt.Sprintf("{\"error\": \"%s\"}", msg)))
}
