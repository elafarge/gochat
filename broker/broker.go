package broker

// Broker represents an abstract broker interface
type Broker interface {
	Subscribe(topic, consumer string) (chan []byte, error)
	Publish(topic string, data []byte) error
	Unsubscribe(topic, consumer string) error
}
