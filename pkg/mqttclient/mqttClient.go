package mqttclient

import (
	"github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

type SubscribeOptions struct {
	Topic             string
	QoS               byte
	OnMessageReceived mqtt.MessageHandler
	Logger            *zap.Logger
}

func Subscribe(connectionOptions *mqtt.ClientOptions, subscribeOptions SubscribeOptions) error {
	connectionOptions.OnConnect = func(client mqtt.Client) {
		logger := subscribeOptions.Logger
		logger.Info("Connected to MQTT Broker")
		logger.Info("Will subscribe to topic", zap.String("topic", subscribeOptions.Topic))
		if token := client.Subscribe(subscribeOptions.Topic, subscribeOptions.QoS, subscribeOptions.OnMessageReceived); token.Wait() && token.Error() != nil {
			logger.Error("Could not subscribe", zap.Error(token.Error()))
		}
	}
	client := mqtt.NewClient(connectionOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
