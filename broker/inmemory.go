package broker

import "fmt"

const (
	// Number of client windows that can be opened with the same user id
	MaxSimultaneousConnections = 3
)

type Channels map[string](chan []byte)

func (c Channels) Add(id string) error {
	if len(c) >= MaxSimultaneousConnections {
		return fmt.Errorf("Number of simultaneous connections reached")
	}

	c[id] = make(chan []byte)
	return nil
}

func (c Channels) Remove(id string) error {
	if _, ok := c[id]; !ok {
		return fmt.Errorf("Channel %s doesn't exist", id)
	}
	close(c[id])
	delete(c, id)
	return nil
}

// InMemoryBroker interface implements a simple in-memory Broker using standard Go maps and channels
type InMemoryBroker struct {
	// Maps consumers to channels
	queues map[string]Channels
}

// NewInMemoryBroker returns an in-memory broker instance
func NewInMemoryBroker() *InMemoryBroker {
	return &InMemoryBroker{
		queues: map[string](Channels){},
	}
}

// HasTopic returns true if the given topic exists, false otherwise
func (b *InMemoryBroker) HasTopic(topic string) bool {
	if _, ok := b.queues[topic]; ok {
		return true
	}
	return false
}

// Subscribe returns a channel to messages destinated to a given consumer
func (b *InMemoryBroker) Subscribe(topic, consumer string) (chan []byte, error) {
	if !b.HasTopic(topic) {
		b.queues[topic] = Channels{}
	}
	if err := b.queues[topic].Add(consumer); err != nil {
		return nil, err
	}

	return b.queues[topic][consumer], nil
}

// Unsubscribe deregisters a consumer from our broker
func (b *InMemoryBroker) Unsubscribe(topic, consumer string) error {
	if !b.HasTopic(topic) {
		return fmt.Errorf("Consumer %s doesn't exist", topic)
	}
	if err := b.queues[topic].Remove(consumer); err != nil {
		return fmt.Errorf("Error removing consumer %s from topic %s: %v", consumer, topic)
	}

	if len(b.queues[topic]) == 0 {
		delete(b.queues, topic)
	}
	return nil
}

// Publish sends a given payload to the approprate topic (consumer)
func (b *InMemoryBroker) Publish(topic string, data []byte) error {
	if !b.HasTopic(topic) {
		return fmt.Errorf("Consumer %s doesn't exist", topic)
	}
	for _, ch := range b.queues[topic] {
		ch <- data
	}
	return nil
}
