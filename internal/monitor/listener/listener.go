package listener

import (
	"context"
	"log/slog"
	"sync"
	"time"

	client "github.com/attestantio/go-eth2-client"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type (
	SlotData struct {
		Received  time.Time
		RootBlock phase0.Root
	}
	Listener struct {
		eventProviderClient client.EventsProvider
		receivals           sync.Map
	}
)

func New(eventProviderClient client.EventsProvider) *Listener {
	return &Listener{
		eventProviderClient: eventProviderClient,
		receivals:           sync.Map{},
	}
}

func (l *Listener) Start(ctx context.Context) error {
	if err := l.eventProviderClient.Events(ctx, []string{"head"}, func(event *v1.Event) {
		slog.With("event", event).Debug("event received")
		data := event.Data.(*v1.HeadEvent)
		l.receivals.Store(data.Slot, SlotData{
			Received:  time.Now(),
			RootBlock: data.Block,
		})
	}); err != nil {
		return err
	}
	return nil
}

func (l *Listener) Receival(slot phase0.Slot) (SlotData, bool) {
	value, ok := l.receivals.Load(slot)
	if !ok {
		return SlotData{}, false
	}
	slotData, ok := value.(SlotData)
	if !ok {
		return SlotData{}, false
	}
	return slotData, true
}
