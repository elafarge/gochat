package gochat

import (
	"flag"

	log "github.com/sirupsen/logrus"
)

// Conf holds all our server config
type Conf struct {
	ListenOn string

	MaxMsgSize int
}

// NewConfFromFlags parses CLI flags and returns a config object
func NewConfFromFlags() (conf *Conf, err error) {
	var (
		listenOn   string
		logLevel   string
		maxMsgSize int
	)

	flag.StringVar(&listenOn, "listen-on", "0.0.0.0:8080", "Inteface:port to listen on")
	flag.StringVar(&logLevel, "log-level", "info", "Logrus log level")
	flag.IntVar(&maxMsgSize, "max-msg-size", 1024, "Websocket messages above this size are dropped")
	flag.Parse()

	// Set log level
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.WithField("ctx", "Config").Panicf("Error parsing logrus log level: %v", err)
	}
	log.SetLevel(level)

	return &Conf{
		ListenOn:   listenOn,
		MaxMsgSize: maxMsgSize,
	}, nil
}
