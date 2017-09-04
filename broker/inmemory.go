package broker

import "fmt"

// InMemoryBroker interface implements a simple in-memory Broker using standard Go maps and channels
type InMemoryBroker struct {
	channels map[string](chan []byte)
}

// NewInMemoryBroker returns an in-memory broker instance
func NewInMemoryBroker() *InMemoryBroker {
	return &InMemoryBroker{
		channels: map[string](chan []byte){},
	}
}

// HasTopic returns true if the given topic exists, false otherwise
func (b *InMemoryBroker) HasTopic(consumer string) bool {
	if _, ok := b.channels[consumer]; ok {
		return true
	}
	return false
}

// Subscribe returns a channel to messages destinated to a given consumer
func (b *InMemoryBroker) Subscribe(consumer string) (chan []byte, error) {
	if !b.HasTopic(consumer) {
		b.channels[consumer] = make(chan []byte)
	}

	return b.channels[consumer], nil
}

// Unsubscribe deregisters a consumer from our broker
func (b *InMemoryBroker) Unsubscribe(consumer string) error {
	if !b.HasTopic(consumer) {
		return fmt.Errorf("Consumer %s doesn't exist", consumer)
	}
	close(b.channels[consumer])
	delete(b.channels, consumer)
	return nil
}

// Publish sends a given payload to the approprate topic (consumer)
func (b *InMemoryBroker) Publish(consumer string, data []byte) error {
	if !b.HasTopic(consumer) {
		return fmt.Errorf("Consumer %s doesn't exist", consumer)
	}
	b.channels[consumer] <- data
	return nil
}
