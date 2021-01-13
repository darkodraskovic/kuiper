package main

import (
	"fmt"
	"time"

	"github.com/emqx/kuiper/common"
	"github.com/emqx/kuiper/xstream/api"
	"github.com/mainflux/mainflux/messaging"
	"github.com/mainflux/mainflux/messaging/nats"
	"github.com/mainflux/senml"
)

type mainfluxConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Channel  string `json:"channel"`
	Subtopic string `json:"subtopic"`
}

type mainfluxSink struct {
	cfg   *mainfluxConfig
	pub   nats.Publisher
	topic string
}

// Configure sink with properties from rule action definition
func (ms *mainfluxSink) Configure(props map[string]interface{}) error {
	cfg := &mainfluxConfig{}

	if err := common.MapToStruct(props, cfg); err != nil {
		return fmt.Errorf("Read properties %v fail with error: %v", props, err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("Property Host is required.")
	}
	if cfg.Port == "" {
		return fmt.Errorf("Property Port is required.")
	}
	if cfg.Channel == "" {
		return fmt.Errorf("Property Channel is required.")
	}

	ms.cfg = cfg

	return nil
}

func (ms *mainfluxSink) Open(ctx api.StreamContext) (err error) {
	logger := ctx.GetLogger()
	logger.Debug("Opening mainflux sink")

	addr := fmt.Sprintf("tcp://%s:%s/", ms.cfg.Host, ms.cfg.Port)
	pub, err := nats.NewPublisher(addr)
	if err != nil {
		return fmt.Errorf("Failed to connect to nats at address %s with error: %v", addr, err)
	}
	ms.pub = pub

	topic := "channels." + ms.cfg.Channel
	if len(ms.cfg.Subtopic) > 0 {
		topic += "." + ms.cfg.Subtopic
	}
	ms.topic = topic

	return
}

// Collect publishes to nats messages transferred to sink
func (ms *mainfluxSink) Collect(ctx api.StreamContext, item interface{}) error {
	logger := ctx.GetLogger()
	logger.Debugf("mainflux sink receive %v", item)

	var msg messaging.Message
	msg.Channel = ms.cfg.Channel
	msg.Subtopic = ms.cfg.Subtopic
	msg.Created = time.Now().Unix()
	msg.Protocol = "tcp"

	rec, ok := item.(senml.Record)
	if !ok {
		return fmt.Errorf("Failed to assert senml record format of %v", item)
	}
	pack := senml.Pack{Records: []senml.Record{rec}}
	paylaod, err := senml.Encode(pack, senml.JSON)
	if err != nil {
		return fmt.Errorf("Failed to encode message to senml format")
	}
	msg.Payload = paylaod
	if err := ms.pub.Publish(ms.topic, msg); err != nil {
		return fmt.Errorf("Failed to publish message to %s", ms.topic)
	}

	return nil
}

func (ms *mainfluxSink) Close(ctx api.StreamContext) error {
	if ms.pub != nil {
		ms.pub.Close()
	}

	return nil
}

// Mainflux exports the constructor used to instantiates the plugin
func Mainflux() api.Sink {
	return &mainfluxSink{}
}
