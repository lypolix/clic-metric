package eventsender

import (
	"clic-metric/internal/domain"
	"clic-metric/internal/lib/logger/sl"
	"clic-metric/internal/storage/postgres"
	"context"
	"log/slog"
	"time"
)


type Sender struct {
	storage *postgres.Storage
	log *slog.Logger
}

func New(storage *postgres.Storage, log *slog.Logger) *Sender {
	return &Sender{
		storage: storage,
		log: log,
	}
}

func (s *Sender) StartProcessEvents(ctx context.Context, handlePeriod time.Duration){
	const op = "services.event-sender.StartProcessEvents"

	log := s.log.With(slog.String("op", op))

	ticker := time.NewTicker(handlePeriod)

	go func() {
		for {
			select {
			case <- ctx.Done():
				log.Info("stopping event processing")
			case <- ticker.C:

			}

			event, err := s.storage.GetNewEvent()
			if err != nil {
				log.Error("failed to get new event", sl.Err(err))
				continue
			}

			if event.ID == 0 {
				log.Debug("no new events")
				continue
			}

			//send event
			s.SendMessage(event)

			if err := s.storage.SetDone(event.ID); err != nil {
				log.Error("failed to set event done", sl.Err(err))
				continue
			}
		}

	}()


}


func (s *Sender) SendMessage(event domain.Event) {
	const op = "services.event-sender.SendMessage"

	log := s.log.With(slog.String("op", op))
	log.Info("sending message", slog.Any("event", event)) //что значит any 
}
//блокирурующие и не блокиующие функции это 