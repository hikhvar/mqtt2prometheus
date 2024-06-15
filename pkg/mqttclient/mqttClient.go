package mqttclient

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type SubscribeOptions struct {
	Topic             string
	QoS               byte
	OnMessageReceived mqtt.MessageHandler
	Logger            log.Logger
}

func Subscribe(connectionOptions *mqtt.ClientOptions, subscribeOptions SubscribeOptions) error {
	oldConnect := connectionOptions.OnConnect
	connectionOptions.OnConnect = func(client mqtt.Client) {
		logger := subscribeOptions.Logger
		oldConnect(client)
		level.Info(logger).Log("msg", "Connected to MQTT Broker")
		level.Info(logger).Log("msg", "Will subscribe to topic", "topic", subscribeOptions.Topic)
		if token := client.Subscribe(subscribeOptions.Topic, subscribeOptions.QoS, subscribeOptions.OnMessageReceived); token.Wait() && token.Error() != nil {
			level.Error(logger).Log("msg", "Could not subscribe", "err", token.Error())
		}
	}
	client := mqtt.NewClient(connectionOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
