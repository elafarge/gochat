package main

import (
	"flag"

	log "github.com/sirupsen/logrus"
)

// ChatRelayConf holds all our server config
type ChatRelayConf struct {
	ListenOn string

	MaxMsgSize int
}

// NewChatRelayConfFromFlags parses CLI flags and returns a config object
func NewChatRelayConfFromFlags() (conf *ChatRelayConf, err error) {
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

	return &ChatRelayConf{
		ListenOn:   listenOn,
		MaxMsgSize: maxMsgSize,
	}, nil
}
