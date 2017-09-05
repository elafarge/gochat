package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

var ProcessedMessages int64

func ConnectAndSpam(urlPrefix string, stopChan chan struct{}, user string, ids []string, rate float64) {
	// Initiate the websocket connection
	c, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s/?user=%s", urlPrefix, user), nil)
	if err != nil {
		log.Errorf("dial: %v", err)
		return
	}
	defer c.Close()

	// Let's listen for incoming messages on the websocket connection
	done := make(chan struct{})
	go func() {
		for {
			if c == nil {
				return
			}
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Errorf("Read error: %v", err)
				return
			}
			ProcessedMessages += 1
			log.Debugf("Received message: %s", message)
			select {
			case <-done:
				return
			default:
				continue
			}
		}
	}()

	// And let's spam our server
	ticker := time.NewTicker(time.Duration(int64(1000000000/rate)) * time.Nanosecond)

	for {
		select {
		case <-stopChan:
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			done <- struct{}{}
			return
		case <-ticker.C:
			c.WriteJSON(map[string]interface{}{
				"sender":    user,
				"recipient": ids[rand.Intn(len(ids))],
				"message":   "See you at the curtain call",
			})
		default:
			time.Sleep(time.Duration(int64(1000000000/rate)) * time.Nanosecond)
			continue
		}
	}
}

func main() {

	var (
		addr     string
		nUsers   int
		rate     float64
		logLevel string
	)

	flag.StringVar(&addr, "endpoint", "127.0.0.1:4691", "Address of the gochat server")
	flag.IntVar(&nUsers, "user-count", 1, "Number of consumers talking in parallel")
	flag.Float64Var(&rate, "rate", 1, "Message rate (per second)")
	flag.StringVar(&logLevel, "log-level", "warn", "Logrus log level")

	flag.Parse()

	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.WithField("ctx", "Config").Panicf("Error parsing logrus log level: %v", err)
	}
	log.SetLevel(level)

	// Let's generate our consuemer ids
	userIDs := make([]string, nUsers)
	for i := 0; i < nUsers; i++ {
		userIDs[i] = uuid.NewV4().String()
	}

	// Let's catch CTRL-C signals to stop our spammer
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	stopChan := make(chan struct{})

	t0 := time.Now()
	// Let's spam our users
	for _, user := range userIDs {
		go ConnectAndSpam(fmt.Sprintf("ws://%s", addr), stopChan, user, userIDs, rate)
	}

	// Blocks until CTRL-C
	<-interrupt

	log.Warnf(
		"Observed rate: %f msg/s (%d messages processed)",
		float64(ProcessedMessages)/time.Since(t0).Seconds(),
		ProcessedMessages,
	)
}
