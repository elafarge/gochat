package broker

import (
	"fmt"
	"sync"
)

const (
	// Number of client windows that can be opened with the same user id
	MaxSimultaneousConnections = 3
)

type Channels struct {
	// Each channel maps to a websocket consumer connection
	channels map[string](chan []byte)

	// Lock for each channel map
	mut sync.RWMutex
}

func NewChannels() *Channels {
	return &Channels{
		channels: make(map[string](chan []byte)),
	}
}

func (c *Channels) Add(id string) (chan []byte, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	if len(c.channels) >= MaxSimultaneousConnections {
		return nil, fmt.Errorf("Number of simultaneous connections reached")
	}

	c.channels[id] = make(chan []byte)
	return c.channels[id], nil
}

func (c *Channels) Remove(id string) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	if _, ok := c.channels[id]; !ok {
		return fmt.Errorf("Channel %s doesn't exist", id)
	}
	close(c.channels[id])
	delete(c.channels, id)
	return nil
}

func (c *Channels) Size() int {
	c.mut.RLock()
	defer c.mut.RUnlock()

	return len(c.channels)
}

func (c *Channels) Publish(message []byte) error {
	c.mut.RLock()
	defer c.mut.RUnlock()

	for _, ch := range c.channels {
		select {
		case ch <- message:
		}
	}

	return nil
}

// InMemoryBroker interface implements a simple in-memory Broker using standard Go maps and channels
type InMemoryBroker struct {
	// Maps consumers to channels
	queues map[string]*Channels

	mut sync.RWMutex
}

// NewInMemoryBroker returns an in-memory broker instance
func NewInMemoryBroker() *InMemoryBroker {
	return &InMemoryBroker{
		queues: make(map[string]*Channels),
	}
}

// hasTopic returns true if the given topic exists, false otherwise
func (b *InMemoryBroker) hasTopic(topic string) bool {
	if _, ok := b.queues[topic]; ok {
		return true
	}
	return false
}

// Subscribe returns a channel to messages destinated to a given consumer
func (b *InMemoryBroker) Subscribe(topic, consumer string) (ch chan []byte, err error) {
	b.mut.Lock()
	defer b.mut.Unlock()
	if !b.hasTopic(topic) {
		b.queues[topic] = NewChannels()
	}

	if ch, err = b.queues[topic].Add(consumer); err != nil {
		return nil, err
	}

	return ch, nil
}

// Unsubscribe deregisters a consumer from our broker
func (b *InMemoryBroker) Unsubscribe(topic, consumer string) error {
	b.mut.Lock()
	defer b.mut.Unlock()

	if !b.hasTopic(topic) {
		return fmt.Errorf("Consumer %s doesn't exist", topic)
	}
	if err := b.queues[topic].Remove(consumer); err != nil {
		return fmt.Errorf("Error removing consumer %s from topic %s: %v", consumer, topic)
	}

	if b.queues[topic].Size() == 0 {
		delete(b.queues, topic)
	}
	return nil
}

// Publish sends a given payload to the approprate topic (consumer)
func (b *InMemoryBroker) Publish(topic string, data []byte) error {
	b.mut.RLock()
	defer b.mut.RUnlock()
	if !b.hasTopic(topic) {
		return fmt.Errorf("Consumer %s doesn't exist", topic)
	}
	return b.queues[topic].Publish(data)
}
