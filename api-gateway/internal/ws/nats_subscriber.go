package ws

import (
	"fmt"
	"log"

	"user-service/pkg/contract"

	"github.com/nats-io/nats.go"
)

func SubscribeUserEvents(nc *nats.Conn, hub *Hub) error {
	eventSubjects := []string{
		contract.SubjectUserEventCreated,
		contract.SubjectUserEventUpdated,
		contract.SubjectUserEventDeleted,
	}

	for _, subject := range eventSubjects {
		currentSubject := subject
		if _, err := nc.Subscribe(currentSubject, func(msg *nats.Msg) {
			hub.Broadcast(msg.Data)
		}); err != nil {
			return fmt.Errorf("subscribe %s: %w", currentSubject, err)
		}

		log.Printf("subscribed to %s", currentSubject)
	}

	return nil
}
