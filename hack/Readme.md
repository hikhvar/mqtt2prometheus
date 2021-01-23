# Hack Scenarios

Required is a MQTT client. I use this: https://github.com/shirou/mqttcli

## Shelly (Metric Per Topic)
The scenario is the feature requested by issue https://github.com/hikhvar/mqtt2prometheus/issues/26.

To start the scenario run:
```bash
docker-compose --env-file shelly.env up
```

To publish a message run:
```bash
mqttcli pub --host localhost -p 1883 -t shellies/living-room/sensor/temperature '15'
```

## DHT22 (Object Per Topic)
The default scenario

To start the scenario run:
```bash
docker-compose --env-file dht22.env up
```

To publish a message run:
```bash
mqttcli pub --host localhost -p 1883 -t v1/devices/me/test -m '{"temperature":"12", "humidity":21}'
```