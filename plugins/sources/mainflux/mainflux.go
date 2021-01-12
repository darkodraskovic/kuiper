package main

import (
	"encoding/json"
	"fmt"

	"github.com/emqx/kuiper/common"
	"github.com/emqx/kuiper/xstream/api"
	"github.com/mainflux/mainflux/messaging"
	"github.com/mainflux/mainflux/messaging/nats"
	"github.com/mainflux/senml"
)

const (
	queue = "kuiper"
)

type mainfluxSourceConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Channel  string `json:"channel"`
	Subtopic string `json:"subtopic"`
}

type mainfluxSource struct {
	consumer chan<- api.SourceTuple
	errCh    chan<- error
	logger   api.Logger
	pubSub   nats.PubSub
	topic    string
}

var _ api.Source = (*mainfluxSource)(nil)

func (s *mainfluxSource) Configure(topic string, props map[string]interface{}) error {
	cfg := &mainfluxSourceConfig{}
	if err := common.MapToStruct(props, cfg); err != nil {
		return fmt.Errorf("Read properties %v fail with error: %v", props, err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("Property Host is required.")
	}
	if cfg.Port == "" {
		return fmt.Errorf("Property Port is required.")
	}

	addr := fmt.Sprintf("tcp://%s:%s/", cfg.Host, cfg.Port)
	pubSub, err := nats.NewPubSub(addr, queue, nil)
	if err != nil {
		return fmt.Errorf("Failed to connect to nats at address %s with error: %v", addr, err)
	}
	s.pubSub = pubSub

	topic = nats.SubjectAllChannels
	if len(cfg.Channel) > 0 {
		topic = "channels." + cfg.Channel
		if len(cfg.Subtopic) > 0 {
			topic += "." + cfg.Subtopic
		}
	}
	s.topic = topic

	return nil
}

func (s *mainfluxSource) Open(ctx api.StreamContext, consumer chan<- api.SourceTuple, errCh chan<- error) {
	logger := ctx.GetLogger()
	logger.Debug("Open mainflux source.")

	err := s.pubSub.Subscribe(s.topic, s.handle)
	if err != nil {
		errCh <- fmt.Errorf("Failed to subscribe to nats topic %s with error: %v", s.topic, err)
		return
	}
	logger.Debugf("Subscribed to nats topic %s.", s.topic)

	s.logger = logger
	s.consumer = consumer
	s.errCh = errCh

	<-ctx.Done()
}

func (s *mainfluxSource) handle(message messaging.Message) error {
	s.logger.Debugf("Received SenML message %v.", message)

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
		// convert struct to map
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
	if s.pubSub != nil {
		s.pubSub.Close()
	}

	return nil
}

func Mainflux() api.Source {
	return &mainfluxSource{}
}
