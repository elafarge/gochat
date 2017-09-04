package broker

// Broker represents an abstract broker interface
type Broker interface {
	Subscribe(consumer string) (chan []byte, error)
	HasTopic(consumer string) bool
	Publish(consumer string, data []byte) error
	Unsubscribe(consumer string) error
}
