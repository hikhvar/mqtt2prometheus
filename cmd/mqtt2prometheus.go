package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"
	kitzap "github.com/go-kit/kit/log/zap"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/hikhvar/mqtt2prometheus/pkg/metrics"
	"github.com/hikhvar/mqtt2prometheus/pkg/mqttclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
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
	logLevelFlag    = zap.LevelFlag("log-level", zap.InfoLevel, "sets the default loglevel (default: \"info\")")
	logEncodingFlag = flag.String(
		"log-format",
		"console",
		"set the desired log output format. Valid values are 'console' and 'json'",
	)
	webConfigFlag = flag.String(
		"web-config-file",
		"",
		"[EXPERIMENTAL] Path to configuration file that can enable TLS or authentication for metric scraping.",
	)
	usePasswordFromFile = flag.Bool(
		"treat-mqtt-password-as-file-name",
		false,
		"treat MQTT2PROM_MQTT_PASSWORD as a secret file path e.g. /var/run/secrets/mqtt-credential",
	)
)

func main() {
	flag.Parse()
	if *versionFlag {
		mustShowVersion()
		os.Exit(0)
	}
	logger := mustSetupLogger()
	defer logger.Sync() //nolint:errcheck
	c := make(chan os.Signal, 1)
	cfg, err := config.LoadConfig(*configFlag)
	if err != nil {
		logger.Fatal("Could not load config", zap.Error(err))
	}

	mqtt_user := os.Getenv("MQTT2PROM_MQTT_USER")
	if mqtt_user != "" {
		cfg.MQTT.User = mqtt_user
	}

	mqtt_password := os.Getenv("MQTT2PROM_MQTT_PASSWORD")
	if *usePasswordFromFile {
		if mqtt_password == "" {
			logger.Fatal("MQTT2PROM_MQTT_PASSWORD is required")
		}
		secret, err := ioutil.ReadFile(mqtt_password)
		if err != nil {
			logger.Fatal("unable to read mqtt password from secret file", zap.Error(err))
		}
		cfg.MQTT.Password = string(secret)
	} else {
		if mqtt_password != "" {
			cfg.MQTT.Password = mqtt_password
		}
	}

	mqttClientOptions := mqtt.NewClientOptions()
	mqttClientOptions.AddBroker(cfg.MQTT.Server).SetCleanSession(true)
	mqttClientOptions.SetAutoReconnect(true)
	mqttClientOptions.SetUsername(cfg.MQTT.User)
	mqttClientOptions.SetPassword(cfg.MQTT.Password)

	if cfg.MQTT.ClientID != "" {
		mqttClientOptions.SetClientID(cfg.MQTT.ClientID)
	} else {
		mqttClientOptions.SetClientID(mustMQTTClientID())
	}

	if cfg.MQTT.ClientCert != "" || cfg.MQTT.ClientKey != "" {
		tlsconfig, err := newTLSConfig(cfg)
		if err != nil {
			logger.Fatal("Invalid tls certificate settings", zap.Error(err))
		}
		mqttClientOptions.SetTLSConfig(tlsconfig)
	}

	collector := metrics.NewCollector(cfg.Cache.Timeout, cfg.Metrics, logger)
	extractor, err := setupExtractor(cfg)
	if err != nil {
		logger.Fatal("could not setup a metric extractor", zap.Error(err))
	}
	ingest := metrics.NewIngest(collector, extractor, cfg.MQTT.DeviceIDRegex)
	mqttClientOptions.SetOnConnectHandler(ingest.OnConnectHandler)
	mqttClientOptions.SetConnectionLostHandler(ingest.ConnectionLostHandler)
	errorChan := make(chan error, 1)

	for {
		err = mqttclient.Subscribe(mqttClientOptions, mqttclient.SubscribeOptions{
			Topic:             cfg.MQTT.TopicPath,
			QoS:               cfg.MQTT.QoS,
			OnMessageReceived: ingest.SetupSubscriptionHandler(errorChan),
			Logger:            logger,
		})
		if err == nil {
			// connected, break loop
			break
		}
		logger.Warn("could not connect to mqtt broker, sleep 10 second", zap.Error(err))
		time.Sleep(10 * time.Second)
	}

	var gatherer prometheus.Gatherer
	if cfg.EnableProfiling {
		gatherer = prometheus.DefaultGatherer
	} else {
		reg := prometheus.NewRegistry()
		reg.MustRegister(ingest.Collector())
		reg.MustRegister(collector)
		gatherer = reg
	}
	http.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	s := &http.Server{
		Addr:    getListenAddress(),
		Handler: http.DefaultServeMux,
	}
	go func() {
		err = web.ListenAndServe(s, *webConfigFlag, setupGoKitLogger(logger))
		if err != nil {
			logger.Fatal("Error while serving http", zap.Error(err))
		}
	}()

	for {
		select {
		case <-c:
			logger.Info("Terminated via Signal. Stop.")
			os.Exit(0)
		case err = <-errorChan:
			logger.Error("Error while processing message", zap.Error(err))
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

func mustSetupLogger() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(*logLevelFlag)
	cfg.Encoding = *logEncodingFlag
	if cfg.Encoding == "console" {
		cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	}
	logger, err := cfg.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build logger: %v", err))
	}

	config.SetProcessContext(logger)
	return logger
}

func setupGoKitLogger(l *zap.Logger) log.Logger {
	return kitzap.NewZapSugarLogger(l, zap.NewAtomicLevelAt(*logLevelFlag).Level())
}

func setupExtractor(cfg config.Config) (metrics.Extractor, error) {
	parser := metrics.NewParser(cfg.Metrics, cfg.JsonParsing.Separator, cfg.Cache.StateDir)
	if cfg.MQTT.ObjectPerTopicConfig != nil {
		switch cfg.MQTT.ObjectPerTopicConfig.Encoding {
		case config.EncodingJSON:
			return metrics.NewJSONObjectExtractor(parser), nil
		default:
			return nil, fmt.Errorf("unsupported object format: %s", cfg.MQTT.ObjectPerTopicConfig.Encoding)
		}
	}
	if cfg.MQTT.MetricPerTopicConfig != nil {
		return metrics.NewMetricPerTopicExtractor(parser, cfg.MQTT.MetricPerTopicConfig.MetricNameRegex), nil
	}
	return nil, fmt.Errorf("no extractor configured")
}

func newTLSConfig(cfg config.Config) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	if cfg.MQTT.CACert != "" {
		pemCerts, err := ioutil.ReadFile(cfg.MQTT.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to load ca_cert file: %w", err)
		}
		certpool.AppendCertsFromPEM(pemCerts)
	}

	cert, err := tls.LoadX509KeyPair(cfg.MQTT.ClientCert, cfg.MQTT.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse client certificate: %w", err)
	}

	return &tls.Config{
		RootCAs:            certpool,
		InsecureSkipVerify: false,
		Certificates:       []tls.Certificate{cert},
	}, nil
}
