package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"

	"github.com/alecthomas/kingpin/v2"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/log/level"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/hikhvar/mqtt2prometheus/pkg/metrics"
	"github.com/hikhvar/mqtt2prometheus/pkg/mqttclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

func main() {
	var (
		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		configFlag = kingpin.Flag(
			"config",
			"config file",
		).Default("config.yaml").String()
		/*
			maxRequests = kingpin.Flag(
				"web.max-requests",
				"Maximum number of parallel scrape requests. Use 0 to disable.",
			).Default("40").Int()
		*/
		usePasswordFromFile = kingpin.Flag(
			"treat-mqtt-password-as-file-name",
			"treat MQTT2PROM_MQTT_PASSWORD as a secret file path e.g. /var/run/secrets/mqtt-credential",
		).Default("false").Bool()
		maxProcs = kingpin.Flag(
			"runtime.gomaxprocs", "The target number of CPUs Go will run on (GOMAXPROCS)",
		).Envar("GOMAXPROCS").Default("1").Int()
		toolkitFlags = kingpinflag.AddFlags(kingpin.CommandLine, ":9641")
	)

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("mqtt2prometheus_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting mqtt2prometheus_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		level.Warn(logger).Log("msg", "MQTT2Prometheus Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
	}
	runtime.GOMAXPROCS(*maxProcs)
	level.Debug(logger).Log("msg", "Go MAXPROCS", "procs", runtime.GOMAXPROCS(0))

	prometheus.MustRegister(
		version.NewCollector("mqtt2prometheus_exporter"),
	)
	c := make(chan os.Signal, 1)
	cfg, err := config.LoadConfig(*configFlag, logger)
	if err != nil {
		level.Error(logger).Log("msg", "Could not load config", "err", err)
		os.Exit(1)
	}

	mqtt_user := os.Getenv("MQTT2PROM_MQTT_USER")
	if mqtt_user != "" {
		cfg.MQTT.User = mqtt_user
	}

	mqtt_password := os.Getenv("MQTT2PROM_MQTT_PASSWORD")
	if *usePasswordFromFile {
		if mqtt_password == "" {
			level.Error(logger).Log("msg", "MQTT2PROM_MQTT_PASSWORD is required")
			os.Exit(1)
		}
		secret, err := ioutil.ReadFile(mqtt_password)
		if err != nil {
			level.Error(logger).Log("msg", "unable to read mqtt password from secret file", "err", err)
			os.Exit(1)
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
			level.Error(logger).Log("msg", "Invalid tls certificate settings", "err", err)
			os.Exit(1)
		}
		mqttClientOptions.SetTLSConfig(tlsconfig)
	}

	collector := metrics.NewCollector(cfg.Cache.Timeout, cfg.Metrics, logger)
	extractor, err := setupExtractor(cfg)
	if err != nil {
		level.Error(logger).Log("msg", "could not setup a metric extractor", "err", err)
		os.Exit(1)
	}
	ingest := metrics.NewIngest(collector, extractor, cfg.MQTT.DeviceIDRegex, logger)
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
		level.Warn(logger).Log("msg", "could not connect to mqtt broker, sleep 10 second", "err", err)
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

	http.Handle(*metricsPath, promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	if *metricsPath != "/" {
		landingConfig := web.LandingConfig{
			Name:        "MQTT2Prometheus Exporter",
			Description: "Prometheus MQTT2Prometheus Exporter",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	server := &http.Server{}

	go func() {
		if err := web.ListenAndServe(server, toolkitFlags, logger); err != nil {
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
	}()

	for {
		select {
		case <-c:
			level.Info(logger).Log("msg", "Terminated via Signal. Stop.")
			os.Exit(0)
		case err = <-errorChan:
			level.Error(logger).Log("msg", "Error while processing message", "err", err)
		}
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
