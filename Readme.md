# MQTT2Prometheus
![](https://github.com/hikhvar/mqtt2prometheus/workflows/tests/badge.svg) ![](https://github.com/hikhvar/mqtt2prometheus/workflows/release/badge.svg)


This exporter translates from MQTT topics to prometheus metrics. The core design is that clients send arbitrary JSON messages
 on the topics. The translation is programmed into the mqtt2prometheus since we often can not change the IoT devices sending 
 the messages. Clients can push 
metrics via MQTT to an MQTT Broker. This exporter subscribes to the broker and
publish the received messages as prometheus metrics. I wrote this exporter to publish
metrics from small embedded sensors based on the NodeMCU to prometheus. The used arduino scetch can be found in the [dht22tomqtt](https://github.com/hikhvar/dht22tomqtt) repository. A local hacking environment with mqtt2prometheus, a MQTT broker and a prometheus server is in the [hack](https://github.com/hikhvar/mqtt2prometheus/tree/master/hack) directory.

## Assumptions about Messages and Topics
This exporter makes some assumptions about the MQTT topics. This exporter assumes that each
client publish the metrics into a dedicated topic. The regular expression ìn the configuration field `mqtt.device_id_regex`
defines how to extract the device ID from the MQTT topic. This allow an arbitrary place of the device ID in the mqtt topic.
For example the [tasmota](https://github.com/arendst/Tasmota) firmware pushes the telemetry data to the topics `tele/<deviceid>/SENSOR`.

Let us assume the default configuration from [#ConfigFile]. A sensor publishes the following message
```json
{"temperature":23.20,"humidity":51.60, "computed": {"heat_index":22.92} }
```

to the MQTT topic `devices/me/livingroom`. This message becomes the following prometheus metrics:

```text
temperature{sensor="livingroom",topic="devices/me/livingroom"} 23.2
heat_index{sensor="livingroom",topic="devices/me/livingroom"} 22.92
humidity{sensor="livingroom",topic="devices/me/livingroom"} 51.6
```

### Tasmota
An example configuration for the tasmota based Gosund SP111 device is given in [examples/gosund_sp111.yaml](examples/gosund_sp111.yaml).

## Build

To build the exporter run:

```bash
make build
```

Only the latest two Go major versions are tested and supported. 

### Docker

#### Use Public Image

To start the public available image run:
```bash
docker run -it -v "$(pwd)/config.yaml:/config.yaml"  -p 8002:8002 docker.pkg.github.com/hikhvar/mqtt2prometheus/mqtt2prometheus:latest 
```

#### Build The Image locally
To build a docker container with the mqtt2prometheus exporter included run:

```bash
make container
```

To run the container with a given config file:

```bash
docker run -it -v "$(pwd)/config.yaml:/config.yaml"  -p 9641:9641 mqtt2prometheus:latest 
```

## Configuration
The exporter can be configured via command line and config file. 

### Commandline
Available command line flags:

```text
Usage of ./mqtt2prometheus.linux_amd64:
  -config string
        config file (default "config.yaml")
  -listen-address string
        listen address for HTTP server used to expose metrics
  -listen-port string
        HTTP port used to expose metrics (default "9641")

```

### Config file
The config file can look like this:

```yaml
# Settings for the MQTT Client. Currently only these three are supported
mqtt:
  # The MQTT broker to connect to
  server: tcp://127.0.0.1:1883
  # The Topic path to subscribe to. Be aware that you have to specify the wildcard. 
  topic_path: v1/devices/me/+
  # Optional: Regular expression to extract the device ID from the topic path. The default regular expression, assumes
  # that the last "element" of the topic_path is the device id.
  # The regular expression must contain a named capture group with the name deviceid
  # For example the expression for tasamota based sensors is "tele/(?P<deviceid>.*)/.*"
  device_id_regex: "(.*/)?(?P<deviceid>.*)"
  # The MQTT QoS level
  qos: 0
cache:
  # Timeout. Each received metric will be presented for this time if no update is send via MQTT
  timeout: 24h
# This is a list of valid metrics. Only metrics listed here will be exported
metrics:
    # The name of the metric in prometheus
  - prom_name: temperature_celsius
    # The name of the metric in a MQTT JSON message. This can be an arbitrary gojsonq path.
    mqtt_name: temperature
    # The prometheus help text for this metric
    help: DHT22 temperature reading
    # The prometheus type for this metric. Valid values are: "gauge" and "counter"
    type: gauge
    # A map of string to string for constant labels. This labels will be attached to every prometheus metric
    const_labels:
      sensor_type: dht22
    # The name of the metric in prometheus
  - prom_name: humidity
    # The name of the metric in a MQTT JSON message
    mqtt_name: humidity
    # The prometheus help text for this metric
    help: DHT22 humidity reading
    # The prometheus type for this metric. Valid values are: "gauge" and "counter"
    type: gauge
    # A map of string to string for constant labels. This labels will be attached to every prometheus metric
    const_labels:
      sensor_type: dht22
    # The name of the metric in prometheus
  - prom_name: heat_index
    # The name of the metric in a MQTT JSON message. Here a nested field.
    mqtt_name: computed.heat_index
    # The prometheus help text for this metric
    help: DHT22 heatIndex calculation
    # The prometheus type for this metric. Valid values are: "gauge" and "counter"
    type: gauge
    # A map of string to string for constant labels. This labels will be attached to every prometheus metric
    const_labels:
      sensor_type: dht22%       
```
