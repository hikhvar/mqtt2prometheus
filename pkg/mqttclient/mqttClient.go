package mqttclient

import (
	"log"

	"github.com/eclipse/paho.mqtt.golang"
)

type SubscribeOptions struct {
	Topic             string
	QoS               byte
	OnMessageReceived mqtt.MessageHandler
}

func Subscribe(connectionOptions *mqtt.ClientOptions, subscribeOptions SubscribeOptions) error {
	connectionOptions.OnConnect = func(client mqtt.Client) {
		log.Print("Connected to MQTT Broker.\n")
		log.Printf("Will subscribe to topic %s", subscribeOptions.Topic)
		if token := client.Subscribe(subscribeOptions.Topic, subscribeOptions.QoS, subscribeOptions.OnMessageReceived); token.Wait() && token.Error() != nil {
			log.Printf("Could not subscribe %s\n", token.Error().Error())
		}
	}
	client := mqtt.NewClient(connectionOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
