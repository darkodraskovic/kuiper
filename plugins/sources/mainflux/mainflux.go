package main

import (
	"encoding/json"
	"fmt"

	"github.com/emqx/kuiper/xstream/api"
	"github.com/mainflux/mainflux/messaging"
	"github.com/mainflux/mainflux/messaging/nats"
	"github.com/mainflux/senml"
)

const (
	queue = "kuiper"
	url   = "nats://localhost:4222"
)

type mainfluxSource struct {
	pubSub   nats.PubSub
	consumer chan<- api.SourceTuple
	errCh    chan<- error
}

var _ api.Source = (*mainfluxSource)(nil)

func (s *mainfluxSource) Configure(topic string, props map[string]interface{}) error {
	var err error
	s.pubSub, err = nats.NewPubSub(url, queue, nil)
	if err != nil {
		return fmt.Errorf("Failed to subscribe to nats with error: %v", err)
	}

	return nil
}

func (s *mainfluxSource) Open(ctx api.StreamContext, consumer chan<- api.SourceTuple, errCh chan<- error) {
	logger := ctx.GetLogger()
	logger.Debug("open mainflux source")

	s.consumer = consumer
	s.errCh = errCh

	err := s.pubSub.Subscribe(nats.SubjectAllChannels, s.handle)
	if err != nil {
		errCh <- err
		return
	}
	defer s.pubSub.Close()

	<-ctx.Done()
}

func (s *mainfluxSource) handle(message messaging.Message) error {
	meta := make(map[string]interface{})
	meta["channel"] = message.Channel
	meta["subtopic"] = message.Subtopic
	meta["publisher"] = message.Publisher
	meta["created"] = message.Created

	pack, err := senml.Decode(message.Payload, senml.JSON)
	if err != nil {
		s.errCh <- err
	}

	for _, rec := range pack.Records {
		recJson, err := json.Marshal(rec)
		if err != nil {
			s.errCh <- err
		}
		recMap := make(map[string]interface{})
		err = json.Unmarshal(recJson, &recMap)
		if err != nil {
			s.errCh <- err
		}

		s.consumer <- api.NewDefaultSourceTuple(recMap, meta)
	}

	return nil
}

func (s *mainfluxSource) Close(ctx api.StreamContext) error {
	s.pubSub.Close()
	return nil
}

func Mainflux() api.Source {
	return &mainfluxSource{}
}
