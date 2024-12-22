# MQTT2Prometheus
![](https://github.com/hikhvar/mqtt2prometheus/workflows/tests/badge.svg) ![](https://github.com/hikhvar/mqtt2prometheus/workflows/release/badge.svg)


This exporter translates from MQTT topics to prometheus metrics. The core design is that clients send arbitrary JSON messages
on the topics. The translation between the MQTT representation and prometheus metrics is configured in the mqtt2prometheus exporter since we often can not change the IoT devices sending 
the messages. Clients can push metrics via MQTT to an MQTT Broker. This exporter subscribes to the broker and
expose the received messages as prometheus metrics. Currently, the exporter supports only MQTT 3.1.

![Overview Diagram](docs/overview.drawio.svg)

I wrote this exporter to expose metrics from small embedded sensors based on the NodeMCU to prometheus.
The used arduino sketch can be found in the [dht22tomqtt](https://github.com/hikhvar/dht22tomqtt) repository. 
A local hacking environment with mqtt2prometheus, a MQTT broker and a prometheus server is in the [hack](https://github.com/hikhvar/mqtt2prometheus/tree/master/hack) directory.

## Assumptions about Messages and Topics
This exporter makes some assumptions about the MQTT topics. This exporter assumes that each
client publish the metrics into a dedicated topic. The regular expression in the configuration field `mqtt.device_id_regex`
defines how to extract the device ID from the MQTT topic. This allows an arbitrary place of the device ID in the mqtt topic.
For example the [tasmota](https://github.com/arendst/Tasmota) firmware pushes the telemetry data to the topics `tele/<deviceid>/SENSOR`.

Let us assume the default configuration from [configuration file](#config-file). A sensor publishes the following message
```json
{"temperature":23.20,"humidity":51.60, "computed": {"heat_index":22.92} }
```

to the MQTT topic `devices/home/livingroom`. This message becomes the following prometheus metrics:

```text
temperature{sensor="livingroom",topic="devices/home/livingroom"} 23.2
heat_index{sensor="livingroom",topic="devices/home/livingroom"} 22.92
humidity{sensor="livingroom",topic="devices/home/livingroom"} 51.6
```

The label `sensor` is extracted with the default `device_id_regex` `(.*/)?(?P<deviceid>.*)` from the MQTT topic `devices/home/livingroom`.
The `device_id_regex` is able to extract exactly one label from the topic path. It extracts only the `deviceid` regex capture group into the `sensor` prometheus label.
To extract more labels from the topic path, have a look at [this FAQ answer](#extract-more-labels-from-the-topic-path).

The topic path can contain multiple wildcards. MQTT has two wildcards: 
* `+`: Single level of hierarchy in the topic path
* `#`: Many levels of hierarchy in the topic path

This [page](https://mosquitto.org/man/mqtt-7.html) explains the wildcard in depth.

For example the `topic_path: devices/+/sensors/#` will match:
* `devices/home/sensors/foo/bar`
* `devices/workshop/sensors/temperature`

### JSON Separator
The exporter interprets `mqtt_name` as [gojsonq](https://github.com/thedevsaddam/gojsonq) paths. Those paths will be used
to find the value in the JSON message.
For example `mqtt_name: computed.heat_index`
addresses
```json
{
  "computed": {
    "heat_index":22.92
  }
}
```
Some sensors might use a `.` in the JSON keys. Therefore, there the configuration option `json_parsing.seperator` in 
the exporter config. This allows us to use any other string to separate hierarchies in the gojsonq path.
E.g let's assume the following MQTT JSON message:
```json
{
  "computed": {
    "heat.index":22.92
  }
}
```
We can now set `json_parsing.seperator` to `/`. This allows us to specify `mqtt_name` as `computed/heat.index`. Keep in mind, 
`json_parsing.seperator` is a global setting. This affects all `mqtt_name` fields in your configuration.

Some devices like Shelly Plus H&T publish one metric per-topic in a JSON format:
```
shellies/shellyplusht-xxx/status/humidity:0 {"id": 0,"rh":51.9}
```
You can use PayloadField to extract the desired value.

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
docker run -it -v "$(pwd)/config.yaml:/config.yaml"  -p  9641:9641 ghcr.io/hikhvar/mqtt2prometheus:latest 
```
Please have a look at the [latest relase](https://github.com/hikhvar/mqtt2prometheus/releases/latest) to get a stable image tag. The latest tag may break at any moment in time since latest is pushed into the registries on every git commit in the master branch. 

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
Usage of ./mqtt2prometheus:
  -config string
        config file (default "config.yaml")
  -listen-address string
        listen address for HTTP server used to expose metrics (default "0.0.0.0")
  -listen-port string
        HTTP port used to expose metrics (default "9641")
  -log-format string
        set the desired log output format. Valid values are 'console' and 'json' (default "console")
  -log-level value
        sets the default loglevel (default: "info")
  -version
        show the builds version, date and commit
  -web-config-file string
        [EXPERIMENTAL] Path to configuration file that can enable TLS or authentication for metric scraping.
  -treat-mqtt-password-as-file-name bool (default: false)
        treat MQTT2PROM_MQTT_PASSWORD environment variable as a secret file path e.g. /var/run/secrets/mqtt-credential. Useful when docker secret or external credential management agents handle the secret file. 
```
The logging is implemented via [zap](https://github.com/uber-go/zap). The logs are printed to `stderr` and valid log levels are
those supported by zap.  


### Config file
The config file can look like this:

```yaml
mqtt:
 # The MQTT broker to connect to
 server: tcp://127.0.0.1:1883
 # Optional: Username and Password for authenticating with the MQTT Server
 user: bob
 password: happylittleclouds
 # Optional: for TLS client certificates
 ca_cert: certs/AmazonRootCA1.pem
 client_cert: certs/xxxxx-certificate.pem.crt
 client_key: certs/xxxxx-private.pem.key
 # Optional: Used to specify ClientID. The default is <hostname>-<pid>
 client_id: somedevice
 # The Topic path to subscribe to. Be aware that you have to specify the wildcard, if you want to follow topics for multiple sensors.
 topic_path: v1/devices/me/+
 # Optional: Regular expression to extract the device ID from the topic path. The default regular expression, assumes
 # that the last "element" of the topic_path is the device id.
 # The regular expression must contain a named capture group with the name deviceid
 # For example the expression for tasamota based sensors is "tele/(?P<deviceid>.*)/.*"
 device_id_regex: "(.*/)?(?P<deviceid>.*)"
 # The MQTT QoS level
 qos: 0
 # NOTE: Only one of metric_per_topic_config or object_per_topic_config should be specified in the configuration
 # Optional: Configures mqtt2prometheus to expect a single metric to be published as the value on an mqtt topic.
 metric_per_topic_config:
  # A regex used for extracting the metric name from the topic. Must contain a named group for `metricname`.
  metric_name_regex: "(.*/)?(?P<metricname>.*)"
 # Optional: Configures mqtt2prometheus to expect an object containing multiple metrics to be published as the value on an mqtt topic.
 # This is the default. 
 object_per_topic_config:
  # The encoding of the object, currently only json is supported
  encoding: JSON
cache:
 # Timeout. Each received metric will be presented for this time if no update is send via MQTT.
 # Set the timeout to -1 to disable the deletion of metrics from the cache. The exporter presents the ingest timestamp
 # to prometheus.
 timeout: 24h
 # Path to the directory to keep the state for monotonic metrics.
 state_directory: "/var/lib/mqtt2prometheus"
json_parsing:
 # Separator. Used to split path to elements when accessing json fields.
 # You can access json fields with dots in it. F.E. {"key.name": {"nested": "value"}}
 # Just set separator to -> and use key.name->nested as mqtt_name
 separator: .
# This is a list of valid metrics. Only metrics listed here will be exported
metrics:
 # The name of the metric in prometheus
 - prom_name: temperature
  # The name of the metric in a MQTT JSON message
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
  # The scale of the metric in a MQTT JSON message (prom_value = mqtt_value * scale)
   mqtt_value_scale: 100
  # The prometheus help text for this metric
   help: DHT22 humidity reading
  # The prometheus type for this metric. Valid values are: "gauge" and "counter"
   type: gauge
  # A map of string to string for constant labels. This labels will be attached to every prometheus metric
   const_labels:
    sensor_type: dht22
  # The name of the metric in prometheus
 - prom_name: heat_index
  # The path of the metric in a MQTT JSON message
   mqtt_name: computed.heat_index
  # The prometheus help text for this metric
   help: DHT22 heatIndex calculation
  # The prometheus type for this metric. Valid values are: "gauge" and "counter"
   type: gauge
  # A map of string to string for constant labels. This labels will be attached to every prometheus metric
   const_labels:
    sensor_type: dht22
  # The name of the metric in prometheus
 - prom_name: state
  # The name of the metric in a MQTT JSON message
   mqtt_name: state
  # Regular expression to only match sensors with the given name pattern
   sensor_name_filter: "^.*-light$"
  # The prometheus help text for this metric
   help: Light state
  # The prometheus type for this metric. Valid values are: "gauge" and "counter"
   type: gauge
  # according to prometheus exposition format timestamp is not mandatory, we can omit it if the reporting from the sensor is sporadic
   omit_timestamp: true
  # A map of string to string for constant labels. This labels will be attached to every prometheus metric
   const_labels:
    sensor_type: ikea
  # When specified, metric value to use if a value cannot be parsed (match cannot be found in the map above, invalid float parsing, expression fails, ...)
  # If not specified, parsing error will occur.
  error_value: 1
  # When specified, enables mapping between string values to metric values.
   string_value_mapping:
    # A map of string to metric value.
    map:
     off: 0
     low: 0
  # The name of the metric in prometheus
 - prom_name: total_light_usage_seconds
  # The name of the metric in a MQTT JSON message
   mqtt_name: state
  # Regular expression to only match sensors with the given name pattern
   sensor_name_filter: "^.*-light$"
  # The prometheus help text for this metric
   help: Total time the light was on, in seconds
  # The prometheus type for this metric. Valid values are: "gauge" and "counter"
   type: counter
  # according to prometheus exposition format timestamp is not mandatory, we can omit it if the reporting from the sensor is sporadic
   omit_timestamp: true
  # A map of string to string for constant labels. This labels will be attached to every prometheus metric
   const_labels:
    sensor_type: ikea
  # Metric value to use if a value cannot be parsed (match cannot be found in the map above, invalid float parsing, ...)
  # If not specified, parsing error will occur.
  error_value: 1
  # When specified, enables mapping between string values to metric values.
   string_value_mapping:
    # A map of string to metric value.
    map:
     off: 0
     low: 0
  # Sum up the time the light is on, see the section "Expressions" below.
  expression: "value > 0 ? last_result + elapsed.Seconds() : last_result"
  # The name of the metric in prometheus
 - prom_name: total_energy
  # The name of the metric in a MQTT JSON message
   mqtt_name: aenergy.total
  # Regular expression to only match sensors with the given name pattern
   sensor_name_filter: "^shellyplus1pm-.*$"
  # The prometheus help text for this metric
   help: Total energy used
  # The prometheus type for this metric. Valid values are: "gauge" and "counter"
   type: counter
  # This setting requires an almost monotonic counter as the source. When monotonicy is enforced, the metric value is regularly written to disk. Thus, resets in the source counter can be detected and corrected by adding an offset as if the reset did not happen. The result is a true monotonic increasing time series, like an ever growing counter.
   force_monotonicy: true
```

### Environment Variables

Having the MQTT login details in the config file runs the risk of publishing them to a version control system. To avoid this, you can supply these parameters via environment variables. MQTT2Prometheus will look for `MQTT2PROM_MQTT_USER` and `MQTT2PROM_MQTT_PASSWORD` in the local environment and load them on startup.

#### Example use with Docker

Create a file to store your login details, for example at `~/secrets/mqtt2prom`:
```SHELL
#!/bin/bash
export MQTT2PROM_MQTT_USER="myUser" 
export MQTT2PROM_MQTT_PASSWORD="superpassword"
```

Then load that file into the environment before starting the container:
```SHELL
 source ~/secrets/mqtt2prom && \
  docker run -it \
  -e MQTT2PROM_MQTT_USER \
  -e MQTT2PROM_MQTT_PASSWORD \
  -v "$(pwd)/examples/config.yaml:/config.yaml" \
  -p 9641:9641 \
  ghcr.io/hikhvar/mqtt2prometheus:latest
```

#### Example use with Docker secret (in swarm)

Create a docker secret to store the password(`mqtt-credential` in the example below), and pass the optional `treat-mqtt-password-as-file-name` command line argument.
```docker
  mqtt_exporter_tasmota:
    image: ghcr.io/hikhvar/mqtt2prometheus:latest 
    secrets:
      - mqtt-credential 
    environment:
      - MQTT2PROM_MQTT_USER=mqtt
      - MQTT2PROM_MQTT_PASSWORD=/var/run/secrets/mqtt-credential
    entrypoint:
      - /mqtt2prometheus
      - -log-level=debug
      - -treat-mqtt-password-as-file-name=true
    volumes:
        - config-tasmota.yml:/config.yaml:ro
```

### Expressions

Metric values can be derived from sensor inputs using complex expressions. Set the metric config option `expression` to the desired formular to calculate the result from the input. Here's an example which integrates all positive values over time:

```yaml
expression: "value > 0 ? last_result + value * elapsed.Seconds() : last_result"
```

During the evaluation, the following variables are available to the expression:
* `value` - the current sensor value (after string-value mapping, if configured)
* `last_value` - the `value` during the previous expression evaluation
* `last_result` - the result from the previous expression evaluation
* `elapsed` - the time that passed since the previous evaluation, as a [Duration](https://pkg.go.dev/time#Duration) value

The [language definition](https://expr-lang.org/docs/v1.9/Language-Definition) describes the expression syntax. In addition, the following functions are available:
* `now()` - the current time as a [Time](https://pkg.go.dev/time#Time) value
* `int(x)` - convert `x` to an integer value
* `float(x)` - convert `x` to a floating point value
* `round(x)` - rounds value `x` to the nearest integer
* `ceil(x)` - rounds value `x` up to the next higher integer
* `floor(x)` - rounds value `x` down to the next lower integer
* `abs(x)` - returns the `x` as a positive number
* `min(x, y)` - returns the minimum of `x` and `y`
* `max(x, y)` - returns the maximum of `x` and `y`

[Time](https://pkg.go.dev/time#Time) and [Duration](https://pkg.go.dev/time#Duration) values come with their own methods which can be used in expressions. For example, `elapsed.Milliseconds()` yields the number of milliseconds that passed since the last evaluation, while `now().Sub(elapsed).Weekday()` returns the day of the week during the previous evaluation.

The `last_value`, `last_result`, and the timestamp of the last evaluation are regularly stored on disk. When mqtt2prometheus is restarted, the data is read back for the next evaluation. This means that you can calculate stable, long-running time serious which depend on the previous result.

#### Evaluation Order

It is important to understand the sequence of transformations from a sensor input to the final output which is exported to Prometheus. The steps are as follows:

1. The sensor input is converted to a number. If a `string_value_mapping` is configured, it is consulted for the conversion.
1. If an `expression` is configured, it is evaluated using the converted number. The result of the evaluation replaces the converted sensor value.
1. If `force_monotonicy` is set to `true`, any new value that is smaller than the previous one is considered to be a counter reset. When a reset is detected, the previous value becomes the value offset which is automatically added to each consecutive value. The offset is persistet between restarts of mqtt2prometheus.
1. If `mqtt_value_scale` is set to a non-zero value, it is applied to the the value to yield the final metric value.

## Frequently Asked Questions

### Listen to multiple Topic Pathes
The exporter can only listen to one topic_path per instance. If you have to listen to two different topic_paths it is 
recommended to run two instances of the mqtt2prometheus exporter. You can run both on the same host or if you run in Kubernetes,
even in the same pod.

### Extract more Labels from the Topic Path
A regular use case is, that user want to extract more labels from the topic path. E.g. they have sensors not only in their `home` but also
in their `workshop` and they encode the location in the topic path. E.g. a sensor pushes the message

```json
{"temperature":3.0,"humidity":34.60, "computed": {"heat_index":15.92} }
```

to the topic `devices/workshop/storage`, this will produce the prometheus metrics with the default configuration.

```text
temperature{sensor="storage",topic="devices/workshop/storage"} 3.0
heat_index{sensor="storage",topic="devices/workshop/storage"} 15.92
humidity{sensor="storage",topic="devices/workshop/storage"} 34.60
```

The following prometheus [relabel_config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#relabel_config) will extract the location from the topic path as well and attaches the `location` label. 
```yaml
relabel_config:
  - source_labels: [ "topic" ]
    target_label: location
    regex: '/devices/(.*)/.*'
    action: replace
    replacement: "$1"
```

With this config added to your prometheus scrape config you will get the following metrics in prometheus storage:

```text
temperature{sensor="storage", location="workshop", topic="devices/workshop/storage"} 3.0
heat_index{sensor="storage", location="workshop", topic="devices/workshop/storage"} 15.92
humidity{sensor="storage", location="workshop", topic="devices/workshop/storage"} 34.60
```
