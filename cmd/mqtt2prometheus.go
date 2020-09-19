package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/hikhvar/mqtt2prometheus/pkg/metrics"
	"github.com/hikhvar/mqtt2prometheus/pkg/mqttclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// These variables are set by goreleaser at linking time.
var (
	version string
	commit  string
	date    string
)

var (
	configFlag = flag.String(
		"config",
		"config.yaml",
		"config file",
	)
	portFlag = flag.String(
		"listen-port",
		"9641",
		"HTTP port used to expose metrics",
	)
	addressFlag = flag.String(
		"listen-address",
		"0.0.0.0",
		"listen address for HTTP server used to expose metrics",
	)
	versionFlag = flag.Bool(
		"version",
		false,
		"show the builds version, date and commit",
	)
)

func main() {
	flag.Parse()
	if *versionFlag {
		mustShowVersion()
		os.Exit(0)
	}
	c := make(chan os.Signal, 1)
	hostName, err := os.Hostname()
	if err != nil {
		log.Fatalf("Could not get hostname. %s\n", err.Error())
	}
	cfg, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Could not load config: %s\n", err.Error())
	}
	mqttClientOptions := mqtt.NewClientOptions()
	mqttClientOptions.AddBroker(cfg.MQTT.Server).SetClientID(hostName).SetCleanSession(true)
	mqttClientOptions.SetUsername(cfg.MQTT.User)
	mqttClientOptions.SetPassword(cfg.MQTT.Password)
	mqttClientOptions.SetClientID(mustMQTTClientID())

	collector := metrics.NewCollector(cfg.Cache.Timeout, cfg.Metrics)
	ingest := metrics.NewIngest(collector, cfg.Metrics, cfg.MQTT.DeviceIDRegex)

	errorChan := make(chan error, 1)

	for {
		err = mqttclient.Subscribe(mqttClientOptions, mqttclient.SubscribeOptions{
			Topic:             cfg.MQTT.TopicPath,
			QoS:               cfg.MQTT.QoS,
			OnMessageReceived: ingest.SetupSubscriptionHandler(errorChan),
		})
		if err == nil {
			// connected, break loop
			break
		}
		log.Printf("Could not connect to mqtt broker %s, sleep 10 second", err.Error())
		time.Sleep(10 * time.Second)
	}

	prometheus.MustRegister(ingest.MessageMetric)
	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err = http.ListenAndServe(getListenAddress(), nil)
		if err != nil {
			log.Fatalf("Error while serving http: %s", err.Error())
		}
	}()

	for {
		select {
		case <-c:
			log.Println("Terminated via Signal. Stop.")
			os.Exit(0)
		case err = <-errorChan:
			log.Printf("Error while processing message. %s", err.Error())
		}
	}
}

func getListenAddress() string {
	return fmt.Sprintf("%s:%s", *addressFlag, *portFlag)
}

func mustShowVersion() {
	versionInfo := struct {
		Version string
		Commit  string
		Date    string
	}{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	err := json.NewEncoder(os.Stdout).Encode(versionInfo)
	if err != nil {
		panic(err)
	}
}

func mustMQTTClientID() string {
	host, err := os.Hostname()
	if err != nil {
		panic(fmt.Sprintf("failed to get hostname: %v", err))
	}
	pid := os.Getpid()
	return fmt.Sprintf("%s-%d", host, pid)
}
